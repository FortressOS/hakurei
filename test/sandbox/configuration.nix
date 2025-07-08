{
  lib,
  pkgs,
  config,
  ...
}:
let
  testProgram = pkgs.callPackage ./tool/package.nix { inherit (config.environment.hakurei.package) version; };
in
{
  users.users = {
    alice = {
      isNormalUser = true;
      description = "Alice Foobar";
      password = "foobar";
      uid = 1000;
    };
  };

  home-manager.users.alice.home.stateVersion = "24.11";

  # Automatically login on tty1 as a normal user:
  services.getty.autologinUser = "alice";

  environment = {
    systemPackages = [
      # For checking seccomp outcome:
      testProgram
    ];

    variables = {
      SWAYSOCK = "/tmp/sway-ipc.sock";
      WLR_RENDERER = "pixman";
    };
  };

  # Automatically configure and start Sway when logging in on tty1:
  programs.bash.loginShellInit = ''
    if [ "$(tty)" = "/dev/tty1" ]; then
      set -e

      mkdir -p ~/.config/sway
      (sed s/Mod4/Mod1/ /etc/sway/config &&
      echo 'output * bg ${pkgs.nixos-artwork.wallpapers.simple-light-gray.gnomeFilePath} fill' &&
      echo 'output Virtual-1 res 1680x1050') > ~/.config/sway/config

      sway --validate
      systemd-cat --identifier=session sway && touch /tmp/sway-exit-ok
    fi
  '';

  programs.sway.enable = true;

  virtualisation.qemu.options = [
    # Need to switch to a different GPU driver than the default one (-vga std) so that Sway can launch:
    "-vga none -device virtio-gpu-pci"

    # Increase performance:
    "-smp 8"
  ];

  environment.hakurei = {
    enable = true;
    stateDir = "/var/lib/hakurei";
    users.alice = 0;

    extraHomeConfig = {
      home.stateVersion = "23.05";
    };

    commonPaths = [
      {
        src = "/var/cache";
        write = true;
      }
    ];

    apps = import ./case pkgs.system lib testProgram;
  };
}
