import json
import shlex

q = shlex.quote
NODE_GROUPS = ["nodes", "floating_nodes"]


def swaymsg(command: str = "", succeed=True, type="command"):
    assert command != "" or type != "command", "Must specify command or type"
    shell = q(f"swaymsg -t {q(type)} -- {q(command)}")
    with machine.nested(
            f"sending swaymsg {shell!r}" + " (allowed to fail)" * (not succeed)
    ):
        ret = (machine.succeed if succeed else machine.execute)(
            f"su - alice -c {shell}"
        )

    # execute also returns a status code, but disregard.
    if not succeed:
        _, ret = ret

    if not succeed and not ret:
        return None

    parsed = json.loads(ret)
    return parsed


def walk(tree):
    yield tree
    for group in NODE_GROUPS:
        for node in tree.get(group, []):
            yield from walk(node)


def wait_for_window(pattern):
    def func(last_chance):
        nodes = (node["name"] for node in walk(swaymsg(type="get_tree")))

        if last_chance:
            nodes = list(nodes)
            machine.log(f"Last call! Current list of windows: {nodes}")

        return any(pattern in name for name in nodes)

    retry(func)


def collect_state_ui(name):
    swaymsg(f"exec fortify ps > '/tmp/{name}.ps'")
    machine.copy_from_vm(f"/tmp/{name}.ps", "")
    swaymsg(f"exec fortify --json ps > '/tmp/{name}.json'")
    machine.copy_from_vm(f"/tmp/{name}.json", "")
    machine.screenshot(name)


def check_state(name, enablements):
    instances = json.loads(machine.succeed("sudo -u alice -i XDG_RUNTIME_DIR=/run/user/1000 fortify --json ps"))
    if len(instances) != 1:
        raise Exception(f"unexpected state length {len(instances)}")
    instance = next(iter(instances.values()))

    config = instance['config']

    command = f"{name}-start"
    if not (config['path'].startswith("/nix/store/")) or not (config['path'].endswith(command)):
        raise Exception(f"unexpected path {config['path']}")

    if len(config['args']) != 1 or config['args'][0] != command:
        raise Exception(f"unexpected args {config['args']}")

    if config['confinement']['enablements'] != enablements:
        raise Exception(f"unexpected enablements {instance['config']['confinement']['enablements']}")


def fortify(command):
    swaymsg(f"exec fortify {command}")


start_all()
machine.wait_for_unit("multi-user.target")

# Run fortify Go tests outside of nix build in the background:
machine.succeed("sudo -u untrusted -i fortify-go-test &> /tmp/go-test &")

# To check fortify's version:
print(machine.succeed("sudo -u alice -i fortify version"))

# Wait for Sway to complete startup:
machine.wait_for_file("/run/user/1000/wayland-1")
machine.wait_for_file("/tmp/sway-ipc.sock")

# Deny unmapped uid:
denyOutput = machine.fail("sudo -u untrusted -i fortify run &>/dev/stdout")
print(denyOutput)
denyOutputVerbose = machine.fail("sudo -u untrusted -i fortify -v run &>/dev/stdout")
print(denyOutputVerbose)

# Fail direct fsu call:
print(machine.fail("sudo -u alice -i fsu"))

# Verify PrintBaseError behaviour:
if denyOutput != "fsu: uid 1001 is not in the fsurc file\n":
    raise Exception(f"unexpected deny output:\n{denyOutput}")
if denyOutputVerbose != "fsu: uid 1001 is not in the fsurc file\nfortify: *cannot obtain uid from fsu: permission denied\n":
    raise Exception(f"unexpected deny verbose output:\n{denyOutputVerbose}")

# Check sandbox outcome:
check_offset = 0
def check_sandbox(name):
    global check_offset
    check_offset += 1
    swaymsg(f"exec script /dev/null -E always -qec check-sandbox-{name}")
    machine.wait_for_file(f"/tmp/fortify.1000/tmpdir/{check_offset}/sandbox-ok", timeout=15)


check_sandbox("preset")
check_sandbox("tty")
check_sandbox("mapuid")

def aid(offset):
    return 1+check_offset+offset


def tmpdir_path(offset, name):
    return f"/tmp/fortify.1000/tmpdir/{aid(offset)}/{name}"


# Start fortify permissive defaults outside Wayland session:
print(machine.succeed("sudo -u alice -i fortify -v run -a 0 touch /tmp/pd-bare-ok"))
machine.wait_for_file("/tmp/fortify.1000/tmpdir/0/pd-bare-ok", timeout=5)

# Verify silent output permissive defaults:
output = machine.succeed("sudo -u alice -i fortify run -a 0 true &>/dev/stdout")
if output != "":
    raise Exception(f"unexpected output\n{output}")

# Verify silent output permissive defaults signal:
def silent_output_interrupt(flags):
    swaymsg("exec foot")
    wait_for_window("alice@machine")
    # aid 0 does not have home-manager
    machine.send_chars(f"exec fortify run {flags}-a 0 sh -c 'export PATH=/run/current-system/sw/bin:$PATH && touch /tmp/pd-silent-ready && sleep infinity' &>/tmp/pd-silent\n")
    machine.wait_for_file("/tmp/fortify.1000/tmpdir/0/pd-silent-ready", timeout=15)
    machine.succeed("rm /tmp/fortify.1000/tmpdir/0/pd-silent-ready")
    machine.send_key("ctrl-c")
    machine.wait_until_fails("pgrep foot", timeout=5)
    machine.wait_until_fails(f"pgrep -u alice -f 'fortify run {flags}-a 0 '", timeout=5)
    output = machine.succeed("cat /tmp/pd-silent && rm /tmp/pd-silent")
    if output != "":
        raise Exception(f"unexpected output\n{output}")


silent_output_interrupt("")
silent_output_interrupt("--dbus ") # this one is especially painful as it maintains a helper
silent_output_interrupt("--wayland -X --dbus --pulse ")

# Verify graceful failure on bad Wayland display name:
print(machine.fail("sudo -u alice -i fortify -v run --wayland true"))

# Start fortify permissive defaults within Wayland session:
fortify('-v run --wayland --dbus notify-send -a "NixOS Tests" "Test notification" "Notification from within sandbox." && touch /tmp/dbus-ok')
machine.wait_for_file("/tmp/dbus-ok", timeout=15)
collect_state_ui("dbus_notify_exited")
machine.succeed("pkill -9 mako")

# Check revert type selection:
fortify("-v run --wayland -X --dbus --pulse -u p0 foot && touch /tmp/p0-exit-ok")
wait_for_window("p0@machine")
print(machine.succeed("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000000"))
fortify("-v run --wayland -X --dbus --pulse -u p1 foot && touch /tmp/p1-exit-ok")
wait_for_window("p1@machine")
print(machine.succeed("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000000"))
machine.send_chars("exit\n")
machine.wait_for_file("/tmp/p1-exit-ok", timeout=15)
# Verify acl is kept alive:
print(machine.succeed("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000000"))
machine.send_chars("exit\n")
machine.wait_for_file("/tmp/p0-exit-ok", timeout=15)
machine.fail("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000000")

# Start app (foot) with Wayland enablement:
swaymsg("exec ne-foot")
wait_for_window(f"u0_a{aid(0)}@machine")
machine.send_chars("clear; wayland-info && touch /tmp/client-ok\n")
machine.wait_for_file(tmpdir_path(0, "client-ok"), timeout=15)
collect_state_ui("foot_wayland")
check_state("ne-foot", 1)
# Verify acl on XDG_RUNTIME_DIR:
print(machine.succeed(f"getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep {aid(0) + 1000000}"))
machine.send_chars("exit\n")
machine.wait_until_fails("pgrep foot", timeout=5)
# Verify acl cleanup on XDG_RUNTIME_DIR:
machine.wait_until_fails(f"getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep {aid(0) + 1000000}", timeout=5)

# Start app (foot) with Wayland enablement from a terminal:
swaymsg("exec foot $SHELL -c '(ne-foot) & sleep 1 && fortify show $(fortify ps --short) && touch /tmp/ps-show-ok && cat'")
wait_for_window(f"u0_a{aid(0)}@machine")
machine.send_chars("clear; wayland-info && touch /tmp/term-ok\n")
machine.wait_for_file(tmpdir_path(0, "term-ok"), timeout=15)
machine.wait_for_file("/tmp/ps-show-ok", timeout=5)
collect_state_ui("foot_wayland_term")
check_state("ne-foot", 1)
machine.send_chars("exit\n")
wait_for_window("foot")
machine.send_key("ctrl-c")
machine.wait_until_fails("pgrep foot", timeout=5)

# Test PulseAudio (fortify does not support PipeWire yet):
swaymsg("exec pa-foot")
wait_for_window(f"u0_a{aid(1)}@machine")
machine.send_chars("clear; pactl info && touch /tmp/pulse-ok\n")
machine.wait_for_file(tmpdir_path(1, "pulse-ok"), timeout=15)
collect_state_ui("pulse_wayland")
check_state("pa-foot", 9)
machine.send_chars("exit\n")
machine.wait_until_fails("pgrep foot", timeout=5)

# Test XWayland (foot does not support X):
swaymsg("exec x11-alacritty")
wait_for_window(f"u0_a{aid(2)}@machine")
machine.send_chars("clear; glinfo && touch /tmp/x11-ok\n")
machine.wait_for_file(tmpdir_path(2, "x11-ok"), timeout=15)
collect_state_ui("alacritty_x11")
check_state("x11-alacritty", 2)
machine.send_chars("exit\n")
machine.wait_until_fails("pgrep alacritty", timeout=5)

# Start app (foot) with direct Wayland access:
swaymsg("exec da-foot")
wait_for_window(f"u0_a{aid(3)}@machine")
machine.send_chars("clear; wayland-info && touch /tmp/direct-ok\n")
collect_state_ui("foot_direct")
machine.wait_for_file(tmpdir_path(3, "direct-ok"), timeout=15)
check_state("da-foot", 1)
# Verify acl on XDG_RUNTIME_DIR:
print(machine.succeed(f"getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep {aid(3) + 1000000}"))
machine.send_chars("exit\n")
machine.wait_until_fails("pgrep foot", timeout=5)
# Verify acl cleanup on XDG_RUNTIME_DIR:
machine.wait_until_fails(f"getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep {aid(3) + 1000000}", timeout=5)

# Test syscall filter:
print(machine.fail("sudo -u alice -i XDG_RUNTIME_DIR=/run/user/1000 strace-failure"))

# Exit Sway and verify process exit status 0:
swaymsg("exit", succeed=False)
machine.wait_for_file("/tmp/sway-exit-ok")

# Print fortify runDir contents:
print(machine.succeed("find /run/user/1000/fortify"))

# Verify go test status:
machine.wait_for_file("/tmp/go-test", timeout=5)
print(machine.succeed("cat /tmp/go-test"))
machine.wait_for_file("/tmp/go-test-ok", timeout=5)
