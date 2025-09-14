{
  lib,
  pkgs,
  config,
  ...
}:
let
  testProgram = pkgs.callPackage ./tool/package.nix { inherit (config.environment.hakurei.package) version; };
  testCases = import ./case pkgs.system lib testProgram;
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

      # For checking pd outcome:
      (pkgs.writeShellScriptBin "check-sandbox-pd" ''
        hakurei -v run hakurei-test \
          -p "/var/tmp/.hakurei-check-ok.0" \
          -t ${toString (builtins.toFile "hakurei-pd-want.json" (builtins.toJSON testCases.pd.want))} \
          -s ${testCases.pd.expectedFilter.${pkgs.system}} "$@"
      '')
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
        type = "bind";
        src = "/var/tmp";
        write = true;
      }
      {
        type = "bind";
        src = "/var/cache";
        write = true;
      }
      {
        type = "overlay";
        dst = "/.hakurei/.ro-store";
        lower = [
          "/nix/.ro-store"
          "/nix/.rw-store/upper"
        ];
      }
      {
        type = "overlay";
        dst = "/.hakurei/store";
        lower = [
          "/nix/.ro-store"
          "/nix/.rw-store/upper"
        ];
        upper = "/tmp/.hakurei-store-rw/upper";
        work = "/tmp/.hakurei-store-rw/work";
      }
    ];

    inherit (testCases) apps;
  };
}
