import json
import shlex

q = shlex.quote
NODE_GROUPS = ["nodes", "floating_nodes"]


def swaymsg(command: str = "", succeed=True, type="command"):
    assert command != "" or type != "command", "Must specify command or type"
    shell = q(f"swaymsg -t {q(type)} -- {q(command)}")
    with machine.nested(f"sending swaymsg {shell!r}" + " (allowed to fail)" * (not succeed)):
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
    swaymsg(f"exec hakurei ps > '/tmp/{name}.ps'")
    machine.copy_from_vm(f"/tmp/{name}.ps", "")
    swaymsg(f"exec hakurei --json ps > '/tmp/{name}.json'")
    machine.copy_from_vm(f"/tmp/{name}.json", "")
    machine.screenshot(name)


def check_state(name, enablements):
    instances = json.loads(machine.succeed("sudo -u alice -i XDG_RUNTIME_DIR=/run/user/1000 hakurei --json ps"))
    if len(instances) != 1:
        raise Exception(f"unexpected state length {len(instances)}")
    instance = next(iter(instances.values()))

    config = instance['config']

    command = f"{name}-start"
    if not (config['path'].startswith("/nix/store/")) or not (config['path'].endswith(command)):
        raise Exception(f"unexpected path {config['path']}")

    if len(config['args']) != 1 or config['args'][0] != command:
        raise Exception(f"unexpected args {config['args']}")

    if config['enablements'] != enablements:
        raise Exception(f"unexpected enablements {instance['config']['enablements']}")


def hakurei(command):
    swaymsg(f"exec hakurei {command}")


start_all()
machine.wait_for_unit("multi-user.target")

# To check hakurei's version:
print(machine.succeed("sudo -u alice -i hakurei version"))

# Wait for Sway to complete startup:
machine.wait_for_file("/run/user/1000/wayland-1")
machine.wait_for_file("/tmp/sway-ipc.sock")

# Run hakurei Go tests outside of nix build in the background:
swaymsg("exec hakurei-test")

# Deny unmapped uid:
denyOutput = machine.fail("sudo -u untrusted -i hakurei run &>/dev/stdout")
print(denyOutput)
denyOutputVerbose = machine.fail("sudo -u untrusted -i hakurei -v run &>/dev/stdout")
print(denyOutputVerbose)

# Fail direct hsu call:
print(machine.fail("sudo -u alice -i hsu"))

# Verify hsu fault behaviour:
if denyOutput != "hsu: uid 1001 is not in the hsurc file\n":
    raise Exception(f"unexpected deny output:\n{denyOutput}")
if denyOutputVerbose != "hsu: uid 1001 is not in the hsurc file\nhakurei: *cannot retrieve user id from setuid wrapper: current user is not in the hsurc file\n":
    raise Exception(f"unexpected deny verbose output:\n{denyOutputVerbose}")

# Verify timeout behaviour:
machine.succeed('sudo -u alice -i hakurei-check-linger-timeout > /var/tmp/linger-stdout 2> /var/tmp/linger-stderr')
linger_stdout = machine.succeed("cat /var/tmp/linger-stdout")
linger_stderr = machine.succeed("cat /var/tmp/linger-stderr")
if linger_stdout != "":
    raise Exception(f"unexpected stdout: {linger_stdout}")
if linger_stderr != "init: timeout exceeded waiting for lingering processes\n":
    raise Exception(f"unexpected stderr: {linger_stderr}")

check_offset = 0


def aid(offset):
    return 1+check_offset+offset


def tmpdir_path(offset, name):
    return f"/tmp/hakurei.0/tmpdir/{aid(offset)}/{name}"


# Start hakurei permissive defaults outside Wayland session:
print(machine.succeed("sudo -u alice -i hakurei -v run -a 0 touch /tmp/pd-bare-ok"))
machine.wait_for_file("/tmp/hakurei.0/tmpdir/0/pd-bare-ok", timeout=5)

# Verify silent output permissive defaults:
output = machine.succeed("sudo -u alice -i hakurei run -a 0 true &>/dev/stdout")
if output != "":
    raise Exception(f"unexpected output\n{output}")

# Verify silent output permissive defaults signal:
def silent_output_interrupt(flags):
    swaymsg("exec foot")
    wait_for_window("alice@machine")
    # aid 0 does not have home-manager
    machine.send_chars(f"exec hakurei run {flags}-a 0 sh -c 'export PATH=/run/current-system/sw/bin:$PATH && touch /tmp/pd-silent-ready && sleep infinity' &>/tmp/pd-silent\n")
    machine.wait_for_file("/tmp/hakurei.0/tmpdir/0/pd-silent-ready", timeout=15)
    machine.succeed("rm /tmp/hakurei.0/tmpdir/0/pd-silent-ready")
    machine.send_key("ctrl-c")
    machine.wait_until_fails("pgrep foot", timeout=5)
    machine.wait_until_fails(f"pgrep -u alice -f 'hakurei run {flags}-a 0 '", timeout=5)
    output = machine.succeed("cat /tmp/pd-silent && rm /tmp/pd-silent")
    if output != "":
        raise Exception(f"unexpected output\n{output}")


silent_output_interrupt("")
silent_output_interrupt("--dbus ") # this one is especially painful as it maintains a helper
silent_output_interrupt("--wayland -X --dbus --pulse ")

# Verify graceful failure on bad Wayland display name:
print(machine.fail("sudo -u alice -i hakurei -v run --wayland true"))

# Start hakurei permissive defaults within Wayland session:
hakurei('-v run --wayland --dbus --dbus-log notify-send -a "NixOS Tests" "Test notification" "Notification from within sandbox." && touch /tmp/dbus-ok')
machine.wait_for_file("/tmp/dbus-ok", timeout=15)
collect_state_ui("dbus_notify_exited")
# not in pid namespace, verify termination
machine.wait_until_fails("pgrep xdg-dbus-proxy")
machine.succeed("pkill -9 mako")

# Check revert type selection:
hakurei("-v run --wayland -X --dbus --pulse -u p0 foot && touch /tmp/p0-exit-ok")
wait_for_window("p0@machine")
print(machine.succeed("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000000"))
hakurei("-v run --wayland -X --dbus --pulse -u p1 foot && touch /tmp/p1-exit-ok")
wait_for_window("p1@machine")
print(machine.succeed("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000000"))
machine.send_chars("exit\n")
machine.wait_for_file("/tmp/p1-exit-ok", timeout=15)
# Verify acl is kept alive:
print(machine.succeed("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000000"))
machine.send_chars("exit\n")
machine.wait_for_file("/tmp/p0-exit-ok", timeout=15)
machine.fail("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000000")

# Check interrupt shim behaviour:
swaymsg("exec sh -c 'ne-foot; echo -n $? > /tmp/monitor-exit-code'")
wait_for_window(f"u0_a{aid(0)}@machine")
machine.succeed("pkill -INT -f 'hakurei -v app '")
machine.wait_until_fails("pgrep foot", timeout=5)
machine.wait_for_file("/tmp/monitor-exit-code")
interrupt_exit_code = int(machine.succeed("cat /tmp/monitor-exit-code"))
if interrupt_exit_code != 230:
    raise Exception(f"unexpected exit code {interrupt_exit_code}")

# Check interrupt shim behaviour immediate termination:
swaymsg("exec sh -c 'ne-foot-immediate; echo -n $? > /tmp/monitor-exit-code'")
wait_for_window(f"u0_a{aid(0)}@machine")
machine.succeed("pkill -INT -f 'hakurei -v app '")
machine.wait_until_fails("pgrep foot", timeout=5)
machine.wait_for_file("/tmp/monitor-exit-code")
interrupt_exit_code = int(machine.succeed("cat /tmp/monitor-exit-code"))
if interrupt_exit_code != 254:
    raise Exception(f"unexpected exit code {interrupt_exit_code}")

# Check shim SIGCONT from unexpected process behaviour:
swaymsg("exec sh -c 'ne-foot &> /tmp/shim-cont-unexpected-pid'")
wait_for_window(f"u0_a{aid(0)}@machine")
machine.succeed("pkill -CONT -f 'hakurei shim'")
machine.succeed("pkill -INT -f 'hakurei -v app '")
machine.wait_until_fails("pgrep foot", timeout=5)
machine.wait_for_file("/tmp/shim-cont-unexpected-pid")
print(machine.succeed('grep "shim: got SIGCONT from unexpected process$" /tmp/shim-cont-unexpected-pid'))

# Start app (foot) with Wayland enablement:
swaymsg("exec ne-foot")
wait_for_window(f"u0_a{aid(0)}@machine")
machine.send_chars("clear; wayland-info && touch /tmp/client-ok\n")
machine.wait_for_file(tmpdir_path(0, "client-ok"), timeout=15)
collect_state_ui("foot_wayland")
check_state("ne-foot", {"wayland": True})
# Verify lack of acl on XDG_RUNTIME_DIR:
machine.fail(f"getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep {aid(0) + 1000000}")
machine.send_chars("exit\n")
machine.wait_until_fails("pgrep foot", timeout=5)
machine.fail(f"getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep {aid(0) + 1000000}", timeout=5)

# Test PulseAudio (hakurei does not support PipeWire yet):
swaymsg("exec pa-foot")
wait_for_window(f"u0_a{aid(1)}@machine")
machine.send_chars("clear; pactl info && touch /tmp/pulse-ok\n")
machine.wait_for_file(tmpdir_path(1, "pulse-ok"), timeout=15)
collect_state_ui("pulse_wayland")
check_state("pa-foot", {"wayland": True, "pulse": True})
machine.send_chars("exit\n")
machine.wait_until_fails("pgrep foot", timeout=5)

# Test XWayland (foot does not support X):
swaymsg("exec x11-alacritty")
wait_for_window(f"u0_a{aid(0)}@machine")
machine.send_chars("clear; glinfo && touch /tmp/x11-ok\n")
machine.wait_for_file(tmpdir_path(0, "x11-ok"), timeout=15)
collect_state_ui("alacritty_x11")
check_state("x11-alacritty", {"x11": True})
machine.send_chars("exit\n")
machine.wait_until_fails("pgrep alacritty", timeout=5)

# Start app (foot) with direct Wayland access:
swaymsg("exec da-foot")
wait_for_window(f"u0_a{aid(3)}@machine")
machine.send_chars("clear; wayland-info && touch /tmp/direct-ok\n")
collect_state_ui("foot_direct")
machine.wait_for_file(tmpdir_path(3, "direct-ok"), timeout=15)
check_state("da-foot", {"wayland": True})
# Verify acl on XDG_RUNTIME_DIR:
print(machine.succeed(f"getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep {aid(3) + 1000000}"))
machine.send_chars("exit\n")
machine.wait_until_fails("pgrep foot", timeout=5)
# Verify acl cleanup on XDG_RUNTIME_DIR:
machine.wait_until_fails(f"getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep {aid(3) + 1000000}", timeout=5)

# Test syscall filter:
print(machine.fail("sudo -u alice -i XDG_RUNTIME_DIR=/run/user/1000 strace-failure"))

# Start app (foot) with Wayland enablement from a terminal:
swaymsg("exec foot $SHELL -c '(ne-foot) & disown && exec $SHELL'")
wait_for_window(f"u0_a{aid(0)}@machine")
machine.send_chars("clear; wayland-info && touch /tmp/term-ok\n")
machine.wait_for_file(tmpdir_path(0, "term-ok"), timeout=15)
machine.send_key("alt-h")
machine.send_chars("clear; hakurei show $(hakurei ps --short) && touch /tmp/ps-show-ok && exec cat\n")
machine.wait_for_file("/tmp/ps-show-ok", timeout=5)
collect_state_ui("foot_wayland_term")
check_state("ne-foot", {"wayland": True})
machine.send_key("alt-l")
machine.send_chars("exit\n")
wait_for_window("alice@machine")
machine.send_key("ctrl-c")
machine.wait_until_fails("pgrep foot", timeout=5)

# Exit Sway and verify process exit status 0:
swaymsg("exit", succeed=False)
machine.wait_for_file("/tmp/sway-exit-ok")

# Print hakurei runDir contents:
print(machine.succeed("find /run/user/1000/hakurei"))

# Verify go test status:
machine.wait_for_file("/tmp/hakurei-test-done")
print(machine.succeed("cat /tmp/hakurei-test.log"))
machine.wait_for_file("/tmp/hakurei-test-ok", timeout=2)
