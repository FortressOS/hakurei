import json
import shlex

q = shlex.quote


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


start_all()
machine.wait_for_unit("multi-user.target")

# To check hakurei's version:
print(machine.succeed("sudo -u alice -i hakurei version"))

# Wait for Sway to complete startup:
machine.wait_for_file("/run/user/1000/wayland-1")
machine.wait_for_file("/tmp/sway-ipc.sock")

# Check seccomp outcome:
swaymsg("exec hakurei run cat")
pid = int(machine.wait_until_succeeds("pgrep -U 1000000 -x cat", timeout=5))
print(machine.succeed(f"hakurei-test filter {pid} c698b081ff957afe17a6d94374537d37f2a63f6f9dd75da7546542407a9e32476ebda3312ba7785d7f618542bcfaf27ca27dcc2dddba852069d28bcfe8cad39a &>/dev/stdout", timeout=5))
machine.succeed(f"kill -TERM {pid}")

# Verify capabilities/securebits in user namespace:
print(machine.succeed("sudo -u alice -i hakurei run capsh --print"))
print(machine.succeed("sudo -u alice -i hakurei run capsh --has-no-new-privs"))
print(machine.fail("sudo -u alice -i hakurei run capsh --has-a=CAP_SYS_ADMIN"))
print(machine.fail("sudo -u alice -i hakurei run capsh --has-b=CAP_SYS_ADMIN"))
print(machine.fail("sudo -u alice -i hakurei run capsh --has-i=CAP_SYS_ADMIN"))
print(machine.fail("sudo -u alice -i hakurei run capsh --has-p=CAP_SYS_ADMIN"))
print(machine.fail("sudo -u alice -i hakurei run umount -R /dev"))

# Check sandbox outcome:
check_offset = 0
def check_sandbox(name):
    global check_offset
    check_offset += 1
    swaymsg(f"exec script /dev/null -E always -qec check-sandbox-{name}")
    machine.wait_for_file(f"/tmp/hakurei.1000/tmpdir/{check_offset}/sandbox-ok", timeout=15)


check_sandbox("preset")
check_sandbox("tty")
check_sandbox("mapuid")
check_sandbox("device")

# Exit Sway and verify process exit status 0:
swaymsg("exit", succeed=False)
machine.wait_for_file("/tmp/sway-exit-ok")

# Print hakurei runDir contents:
print(machine.succeed("find /run/user/1000/hakurei"))
