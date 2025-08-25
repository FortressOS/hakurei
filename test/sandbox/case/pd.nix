{
  fs,
  ent,
  ignore,
  ...
}:
{
  # 0, PresetExt | PresetDenyDevel
  expectedFilter = {
    x86_64-linux = "c698b081ff957afe17a6d94374537d37f2a63f6f9dd75da7546542407a9e32476ebda3312ba7785d7f618542bcfaf27ca27dcc2dddba852069d28bcfe8cad39a";
    aarch64-linux = "433ce9b911282d6dcc8029319fb79b816b60d5a795ec8fc94344dd027614d68f023166a91bb881faaeeedd26e3d89474e141e5a69a97e93b8984ca8f14999980";
  };

  want = {
    env = [
      "HOME=/var/lib/hakurei/u0/a0"
      "SHELL=/run/current-system/sw/bin/bash"
      "TERM=linux"
      "USER=u0_a0"
      "XDG_RUNTIME_DIR=/run/user/65534"
      "XDG_SESSION_CLASS=user"
      "XDG_SESSION_TYPE=tty"
    ];

    fs = fs "dead" {
      ".hakurei" = fs "800001ed" { } null;
      bin = fs "800001ed" { sh = fs "80001ff" null null; } null;
      dev = fs "800001ed" {
        console = fs "4200190" null null;
        core = fs "80001ff" null null;
        fd = fs "80001ff" null null;
        full = fs "42001b6" null null;
        kvm = fs "42001b6" null null;
        mqueue = fs "801001ff" { } null;
        null = fs "42001b6" null "";
        ptmx = fs "80001ff" null null;
        pts = fs "800001ed" { ptmx = fs "42001b6" null null; } null;
        random = fs "42001b6" null null;
        shm = fs "801001ff" { } null;
        stderr = fs "80001ff" null null;
        stdin = fs "80001ff" null null;
        stdout = fs "80001ff" null null;
        tty = fs "42001b6" null null;
        urandom = fs "42001b6" null null;
        zero = fs "42001b6" null null;
      } null;
      etc = fs "800001ed" {
        ".clean" = fs "80001ff" null null;
        ".host" = fs "800001c0" null null;
        ".updated" = fs "80001ff" null null;
        "NIXOS" = fs "80001ff" null null;
        "X11" = fs "80001ff" null null;
        "alsa" = fs "80001ff" null null;
        "bash_logout" = fs "80001ff" null null;
        "bashrc" = fs "80001ff" null null;
        "binfmt.d" = fs "80001ff" null null;
        "dbus-1" = fs "80001ff" null null;
        "default" = fs "80001ff" null null;
        "dhcpcd.exit-hook" = fs "80001ff" null null;
        "fonts" = fs "80001ff" null null;
        "fstab" = fs "80001ff" null null;
        "hsurc" = fs "80001ff" null null;
        "fuse.conf" = fs "80001ff" null null;
        "group" = fs "180" null "hakurei:x:65534:\n";
        "host.conf" = fs "80001ff" null null;
        "hostname" = fs "80001ff" null null;
        "hosts" = fs "80001ff" null null;
        "inputrc" = fs "80001ff" null null;
        "issue" = fs "80001ff" null null;
        "kbd" = fs "80001ff" null null;
        "locale.conf" = fs "80001ff" null null;
        "login.defs" = fs "80001ff" null null;
        "lsb-release" = fs "80001ff" null null;
        "lvm" = fs "80001ff" null null;
        "machine-id" = fs "80001ff" null null;
        "man_db.conf" = fs "80001ff" null null;
        "modprobe.d" = fs "80001ff" null null;
        "modules-load.d" = fs "80001ff" null null;
        "mtab" = fs "80001ff" null null;
        "nanorc" = fs "80001ff" null null;
        "netgroup" = fs "80001ff" null null;
        "nix" = fs "80001ff" null null;
        "nixos" = fs "80001ff" null null;
        "nscd.conf" = fs "80001ff" null null;
        "nsswitch.conf" = fs "80001ff" null null;
        "os-release" = fs "80001ff" null null;
        "pam" = fs "80001ff" null null;
        "pam.d" = fs "80001ff" null null;
        "passwd" = fs "180" null "u0_a0:x:65534:65534:Hakurei:/var/lib/hakurei/u0/a0:/run/current-system/sw/bin/bash\n";
        "pipewire" = fs "80001ff" null null;
        "pki" = fs "80001ff" null null;
        "polkit-1" = fs "80001ff" null null;
        "profile" = fs "80001ff" null null;
        "protocols" = fs "80001ff" null null;
        "resolv.conf" = fs "80001ff" null null;
        "resolvconf.conf" = fs "80001ff" null null;
        "rpc" = fs "80001ff" null null;
        "services" = fs "80001ff" null null;
        "set-environment" = fs "80001ff" null null;
        "shadow" = fs "80001ff" null null;
        "shells" = fs "80001ff" null null;
        "ssh" = fs "80001ff" null null;
        "ssl" = fs "80001ff" null null;
        "static" = fs "80001ff" null null;
        "subgid" = fs "80001ff" null null;
        "subuid" = fs "80001ff" null null;
        "sudoers" = fs "80001ff" null null;
        "sway" = fs "80001ff" null null;
        "sysctl.d" = fs "80001ff" null null;
        "systemd" = fs "80001ff" null null;
        "terminfo" = fs "80001ff" null null;
        "tmpfiles.d" = fs "80001ff" null null;
        "udev" = fs "80001ff" null null;
        "vconsole.conf" = fs "80001ff" null null;
        "xdg" = fs "80001ff" null null;
        "zoneinfo" = fs "80001ff" null null;
      } null;
      home = fs "800001ed" { alice = fs "800001c0" null null; } null;
      lib64 = fs "800001ed" { "ld-linux-x86-64.so.2" = fs "80001ff" null null; } null;
      "lost+found" = fs "800001c0" null null;
      nix = fs "800001ed" {
        ".ro-store" = fs "801001fd" null null;
        ".rw-store" = fs "800001ed" null null;
        store = fs "801001fd" null null;
        var = fs "800001ed" {
          log = fs "800001ed" null null;
          nix = fs "800001ed" null null;
        } null;
      } null;
      proc = fs "8000016d" null null;
      root = fs "800001c0" null null;
      run = fs "800001ed" null null;
      srv = fs "800001ed" { } null;
      sys = fs "8000016d" null null;
      tmp = fs "800001f8" { } null;
      usr = fs "800001ed" { bin = fs "800001ed" { env = fs "80001ff" null null; } null; } null;
      var = fs "800001ed" null null;
    } null;

    mount = [
      (ent "/sysroot" "/" "ro,nosuid,nodev,relatime" "tmpfs" "rootfs" "rw,uid=1000000,gid=1000000")
      (ent "/bin" "/bin" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/home" "/home" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/lib64" "/lib64" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/lost+found" "/lost+found" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/nix" "/nix" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/" "/nix/.ro-store" "rw,nosuid,nodev,relatime" "9p" "nix-store" ignore)
      (ent "/" "/nix/.rw-store" "rw,nosuid,nodev,relatime" "tmpfs" "tmpfs" "rw,mode=755")
      (ent "/" "/nix/store" "rw,relatime" "overlay" "overlay" "rw,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work,uuid=on")
      (ent "/" "/nix/store" "ro,nosuid,nodev,relatime" "overlay" "overlay" "rw,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work,uuid=on")
      (ent "/root" "/root" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/" "/run" "rw,nosuid,nodev" "tmpfs" "tmpfs" ignore)
      (ent "/" "/run/keys" "rw,nosuid,nodev,relatime" "ramfs" "ramfs" "rw,mode=750")
      (ent "/" "/run/credentials/systemd-journald.service" "ro,nosuid,nodev,noexec,relatime,nosymfollow" "tmpfs" "tmpfs" "rw,size=1024k,nr_inodes=1024,mode=700,noswap")
      (ent "/" "/run/wrappers" "rw,nosuid,nodev,relatime" "tmpfs" "tmpfs" ignore)
      (ent "/" "/run/credentials/getty@tty1.service" "ro,nosuid,nodev,noexec,relatime,nosymfollow" "tmpfs" "tmpfs" "rw,size=1024k,nr_inodes=1024,mode=700,noswap")
      (ent "/" "/run/user/1000" "rw,nosuid,nodev,relatime" "tmpfs" "tmpfs" ignore)
      (ent "/srv" "/srv" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/" "/sys" "rw,nosuid,nodev,noexec,relatime" "sysfs" "sysfs" "rw")
      (ent "/" "/sys/kernel/security" "rw,nosuid,nodev,noexec,relatime" "securityfs" "securityfs" "rw")
      (ent "/../../.." "/sys/fs/cgroup" "rw,nosuid,nodev,noexec,relatime" "cgroup2" "cgroup2" "rw,nsdelegate,memory_recursiveprot")
      (ent "/" "/sys/fs/pstore" "rw,nosuid,nodev,noexec,relatime" "pstore" "pstore" "rw")
      (ent "/" "/sys/fs/bpf" "rw,nosuid,nodev,noexec,relatime" "bpf" "bpf" "rw,mode=700")
      # systemd race: tracefs debugfs configfs fusectl
      (ent "/" ignore "rw,nosuid,nodev,noexec,relatime" ignore ignore "rw")
      (ent "/" ignore "rw,nosuid,nodev,noexec,relatime" ignore ignore "rw")
      (ent "/" ignore "rw,nosuid,nodev,noexec,relatime" ignore ignore "rw")
      (ent "/" ignore "rw,nosuid,nodev,noexec,relatime" ignore ignore "rw")
      (ent "/usr" "/usr" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/var" "/var" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/" "/proc" "rw,nosuid,nodev,noexec,relatime" "proc" "proc" "rw")
      (ent "/" "/.hakurei" "rw,nosuid,nodev,relatime" "tmpfs" "ephemeral" "rw,size=4k,mode=755,uid=1000000,gid=1000000")
      (ent "/" "/dev" "ro,nosuid,nodev,relatime" "tmpfs" "devtmpfs" "rw,mode=755,uid=1000000,gid=1000000")
      (ent "/null" "/dev/null" "rw,nosuid" "devtmpfs" "devtmpfs" ignore)
      (ent "/zero" "/dev/zero" "rw,nosuid" "devtmpfs" "devtmpfs" ignore)
      (ent "/full" "/dev/full" "rw,nosuid" "devtmpfs" "devtmpfs" ignore)
      (ent "/random" "/dev/random" "rw,nosuid" "devtmpfs" "devtmpfs" ignore)
      (ent "/urandom" "/dev/urandom" "rw,nosuid" "devtmpfs" "devtmpfs" ignore)
      (ent "/tty" "/dev/tty" "rw,nosuid" "devtmpfs" "devtmpfs" ignore)
      (ent "/" "/dev/pts" "rw,nosuid,noexec,relatime" "devpts" "devpts" "rw,mode=620,ptmxmode=666")
      (ent ignore "/dev/console" "rw,nosuid,noexec,relatime" "devpts" "devpts" "rw,gid=3,mode=620,ptmxmode=666")
      (ent "/" "/dev/mqueue" "rw,nosuid,nodev,noexec,relatime" "mqueue" "mqueue" "rw")
      (ent "/kvm" "/dev/kvm" "rw,nosuid" "devtmpfs" "devtmpfs" ignore)
      (ent "/" "/run/nscd" "ro,nosuid,nodev,relatime" "tmpfs" "readonly" "ro,mode=755,uid=1000000,gid=1000000")
      (ent "/etc" ignore "ro,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/" "/run/user/1000" "rw,nosuid,nodev,relatime" "tmpfs" "ephemeral" "rw,size=8k,mode=755,uid=1000000,gid=1000000")
      (ent "/" "/run/dbus" "rw,nosuid,nodev,relatime" "tmpfs" "ephemeral" "rw,size=8k,mode=755,uid=1000000,gid=1000000")
      (ent "/" "/dev/shm" "rw,nosuid,nodev,relatime" "tmpfs" "ephemeral" "rw,uid=1000000,gid=1000000")
      (ent "/" "/run/user" "rw,nosuid,nodev,relatime" "tmpfs" "ephemeral" "rw,size=4k,mode=755,uid=1000000,gid=1000000")
      (ent "/tmp/hakurei.0/runtime/0" "/run/user/65534" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent "/tmp/hakurei.0/tmpdir/0" "/tmp" "rw,nosuid,nodev,relatime" "ext4" "/dev/disk/by-label/nixos" "rw")
      (ent ignore "/etc/passwd" "ro,nosuid,nodev,relatime" "tmpfs" "rootfs" "rw,uid=1000000,gid=1000000")
      (ent ignore "/etc/group" "ro,nosuid,nodev,relatime" "tmpfs" "rootfs" "rw,uid=1000000,gid=1000000")
    ];

    seccomp = true;

    try_socket = "/tmp/.X11-unix/X0";
    socket_abstract = true;
    socket_pathname = false;
  };
}
