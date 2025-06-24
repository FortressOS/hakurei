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

    if len(config['args']) != 1 or not (config['args'][0].startswith("/nix/store/")) or f"hakurei-{name}-" not in (config['args'][0]):
        raise Exception(f"unexpected args {instance['config']['args']}")

    if config['enablements'] != enablements:
        raise Exception(f"unexpected enablements {instance['config']['enablements']}")


start_all()
machine.wait_for_unit("multi-user.target")

# To check hakurei's version:
print(machine.succeed("sudo -u alice -i hakurei version"))

# Wait for Sway to complete startup:
machine.wait_for_file("/run/user/1000/wayland-1")
machine.wait_for_file("/tmp/sway-ipc.sock")

# Prepare fpkg directory:
machine.succeed("install -dm 0700 -o alice -g users /var/lib/hakurei/1000")

# Install fpkg app:
swaymsg("exec fpkg -v install /etc/foot.pkg && touch /tmp/fpkg-install-done")
machine.wait_for_file("/tmp/fpkg-install-done")

# Start app (foot) with Wayland enablement:
swaymsg("exec fpkg -v start org.codeberg.dnkl.foot")
wait_for_window("hakurei@machine-foot")
machine.send_chars("clear; wayland-info && touch /tmp/success-client\n")
machine.wait_for_file("/tmp/hakurei.1000/tmpdir/2/success-client")
collect_state_ui("app_wayland")
check_state("foot", 13)
# Verify acl on XDG_RUNTIME_DIR:
print(machine.succeed("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000002"))
machine.send_chars("exit\n")
machine.wait_until_fails("pgrep foot")
# Verify acl cleanup on XDG_RUNTIME_DIR:
machine.wait_until_fails("getfacl --absolute-names --omit-header --numeric /run/user/1000 | grep 1000002")

# Exit Sway and verify process exit status 0:
swaymsg("exit", succeed=False)
machine.wait_for_file("/tmp/sway-exit-ok")

# Print hakurei runDir contents:
print(machine.succeed("find /run/user/1000/hakurei"))