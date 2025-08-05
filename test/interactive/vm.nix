{
  virtualisation.vmVariant.virtualisation = {
    memorySize = 4096;
    qemu.options = [
      "-vga none -device virtio-gpu-pci"
      "-smp 8"
    ];

    mountHostNixStore = true;
    writableStore = true;
    writableStoreUseTmpfs = false;

    sharedDirectories = {
      cwd = {
        target = "/mnt/.ro-cwd";
        source = ''"$OLDPWD"'';
        securityModel = "none";
      };
    };

    fileSystems = {
      "/mnt/.ro-cwd".options = [
        "ro"
        "noatime"
      ];
      "/mnt/cwd".overlay = {
        lowerdir = [ "/mnt/.ro-cwd" ];
        upperdir = "/tmp/.cwd/upper";
        workdir = "/tmp/.cwd/work";
      };

      "/mnt/src".overlay = {
        lowerdir = [ ../.. ];
        upperdir = "/tmp/.src/upper";
        workdir = "/tmp/.src/work";
      };
    };
  };

  systemd.services = {
    logrotate-checkconf.enable = false;
    hakurei-src-fix-ownership = {
      wantedBy = [ "multi-user.target" ];
      wants = [ "mnt-src.mount" ];
      after = [ "mnt-src.mount" ];
      serviceConfig = {
        Type = "oneshot";
        RemainAfterExit = true;
      };
      script = ''
        chown -R alice:users /mnt/src/
      '';
    };
  };
}
