{
  writeText,
  buildGoModule,

  version,
}:
let
  wantMounts =
    let
      ent = fsname: dir: type: opts: freq: passno: {
        inherit
          fsname
          dir
          type
          opts
          freq
          passno
          ;
      };
    in
    [
      (ent "tmpfs" "/" "tmpfs" "rw,nosuid,nodev,relatime,uid=1000001,gid=1000001" 0 0)
      (ent "proc" "/proc" "proc" "rw,nosuid,nodev,noexec,relatime" 0 0)
      (ent "tmpfs" "/.fortify" "tmpfs" "rw,nosuid,nodev,relatime,size=4k,mode=755,uid=1000001,gid=1000001" 0 0)
      (ent "tmpfs" "/dev" "tmpfs" "rw,nosuid,nodev,relatime,mode=755,uid=1000001,gid=1000001" 0 0)
      (ent "devtmpfs" "/dev/null" "devtmpfs" "host_passthrough" 0 0)
      (ent "devtmpfs" "/dev/zero" "devtmpfs" "host_passthrough" 0 0)
      (ent "devtmpfs" "/dev/full" "devtmpfs" "host_passthrough" 0 0)
      (ent "devtmpfs" "/dev/random" "devtmpfs" "host_passthrough" 0 0)
      (ent "devtmpfs" "/dev/urandom" "devtmpfs" "host_passthrough" 0 0)
      (ent "devtmpfs" "/dev/tty" "devtmpfs" "host_passthrough" 0 0)
      (ent "devpts" "/dev/pts" "devpts" "rw,nosuid,noexec,relatime,mode=620,ptmxmode=666" 0 0)
      (ent "mqueue" "/dev/mqueue" "mqueue" "rw,relatime" 0 0)
      (ent "/dev/disk/by-label/nixos" "/bin" "ext4" "ro,nosuid,nodev,relatime" 0 0)
      (ent "/dev/disk/by-label/nixos" "/usr/bin" "ext4" "ro,nosuid,nodev,relatime" 0 0)
      (ent "overlay" "/nix/store" "overlay" "ro,nosuid,nodev,relatime,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work,uuid=on" 0 0)
      (ent "overlay" "/run/current-system" "overlay" "ro,nosuid,nodev,relatime,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work,uuid=on" 0 0)
      (ent "sysfs" "/sys/block" "sysfs" "ro,nosuid,nodev,noexec,relatime" 0 0)
      (ent "sysfs" "/sys/bus" "sysfs" "ro,nosuid,nodev,noexec,relatime" 0 0)
      (ent "sysfs" "/sys/class" "sysfs" "ro,nosuid,nodev,noexec,relatime" 0 0)
      (ent "sysfs" "/sys/dev" "sysfs" "ro,nosuid,nodev,noexec,relatime" 0 0)
      (ent "sysfs" "/sys/devices" "sysfs" "ro,nosuid,nodev,noexec,relatime" 0 0)
      (ent "overlay" "/run/opengl-driver" "overlay" "ro,nosuid,nodev,relatime,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work,uuid=on" 0 0)
      (ent "devtmpfs" "/dev/dri" "devtmpfs" "host_passthrough" 0 0)
      (ent "proc" "/.fortify/host-mounts" "proc" "ro,nosuid,nodev,noexec,relatime" 0 0)
      (ent "/dev/disk/by-label/nixos" "/.fortify/etc" "ext4" "ro,nosuid,nodev,relatime" 0 0)
      (ent "tmpfs" "/run/user" "tmpfs" "rw,nosuid,nodev,relatime,size=1024k,mode=755,uid=1000001,gid=1000001" 0 0)
      (ent "tmpfs" "/run/user/65534" "tmpfs" "rw,nosuid,nodev,relatime,size=8192k,mode=755,uid=1000001,gid=1000001" 0 0)
      (ent "/dev/disk/by-label/nixos" "/tmp" "ext4" "rw,nosuid,nodev,relatime" 0 0)
      (ent "/dev/disk/by-label/nixos" "/var/lib/fortify/u0/a1" "ext4" "rw,nosuid,nodev,relatime" 0 0)
      (ent "tmpfs" "/etc/passwd" "tmpfs" "ro,nosuid,nodev,relatime,uid=1000001,gid=1000001" 0 0)
      (ent "tmpfs" "/etc/group" "tmpfs" "ro,nosuid,nodev,relatime,uid=1000001,gid=1000001" 0 0)
      (ent "/dev/disk/by-label/nixos" "/run/user/65534/wayland-0" "ext4" "ro,nosuid,nodev,relatime" 0 0)
      (ent "tmpfs" "/run/user/65534/pulse/native" "tmpfs" "host_passthrough" 0 0)
      (ent "/dev/disk/by-label/nixos" "/run/user/65534/bus" "ext4" "ro,nosuid,nodev,relatime" 0 0)
      (ent "tmpfs" "/var/run/nscd" "tmpfs" "rw,nosuid,nodev,relatime,size=8k,mode=755,uid=1000001,gid=1000001" 0 0)
      (ent "overlay" "/.fortify/sbin/fortify" "overlay" "ro,nosuid,nodev,relatime,lowerdir=/mnt-root/nix/.ro-store,upperdir=/mnt-root/nix/.rw-store/upper,workdir=/mnt-root/nix/.rw-store/work,uuid=on" 0 0)
    ];

  mainFile = writeText "main.go" ''
    package main

    import "git.gensokyo.uk/security/fortify/test/sandbox"

    func main() { sandbox.MustAssertMounts("", "/.fortify/host-mounts", "${writeText "want-mounts.json" (builtins.toJSON wantMounts)}") }
  '';
in
buildGoModule {
  pname = "check-mounts";
  inherit version;

  src = ../.;
  vendorHash = null;

  preBuild = ''
    go mod init git.gensokyo.uk/security/fortify/test >& /dev/null
    cp ${mainFile} main.go
  '';
}
