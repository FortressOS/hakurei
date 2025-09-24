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


def check_filter(check_offset, name, pname):
    pid = int(machine.wait_until_succeeds(f"pgrep -U {1000000+check_offset} -x {pname}", timeout=60))
    hash = machine.succeed(f"sudo -u alice -i XDG_RUNTIME_DIR=/run/user/1000 WAYLAND_DISPLAY=wayland-1 check-sandbox-{name} hash")
    print(machine.succeed(f"hakurei-test -s {hash} filter {pid}"))


start_all()
machine.wait_for_unit("multi-user.target")

# To check hakurei's version:
print(machine.succeed("sudo -u alice -i hakurei version"))

# Wait for Sway to complete startup:
machine.wait_for_file("/run/user/1000/wayland-1")
machine.wait_for_file("/tmp/sway-ipc.sock")

# Check pd seccomp outcome:
swaymsg("exec hakurei run cat")
check_filter(0, "pdlike", "cat")

# Verify capabilities/securebits in user namespace:
print(machine.succeed("sudo -u alice -i hakurei run capsh --print"))
print(machine.succeed("sudo -u alice -i hakurei run capsh --has-no-new-privs"))
print(machine.fail("sudo -u alice -i hakurei run capsh --has-a=CAP_SYS_ADMIN"))
print(machine.fail("sudo -u alice -i hakurei run capsh --has-b=CAP_SYS_ADMIN"))
print(machine.fail("sudo -u alice -i hakurei run capsh --has-i=CAP_SYS_ADMIN"))
print(machine.fail("sudo -u alice -i hakurei run capsh --has-p=CAP_SYS_ADMIN"))
print(machine.fail("sudo -u alice -i hakurei run umount -R /dev"))

# Check sandbox outcome:
machine.succeed("install -dm0777 /tmp/.hakurei-store-rw/{upper,work}")
check_offset = 0
def check_sandbox(name):
    global check_offset
    swaymsg(f"exec script /dev/null -E always -qec check-sandbox-{name}")
    machine.wait_for_file(f"/var/tmp/.hakurei-check-ok.{check_offset}", timeout=60)
    check_filter(check_offset, name, "hakurei-test")
    check_offset += 1


check_sandbox("pd")
check_sandbox("preset")
check_sandbox("tty")
check_sandbox("mapuid")
check_sandbox("device")
check_sandbox("pdlike")

# Exit Sway and verify process exit status 0:
swaymsg("exit", succeed=False)
machine.wait_for_file("/tmp/sway-exit-ok")

# Print hakurei runDir contents:
print(machine.succeed("find /run/user/1000/hakurei"))
