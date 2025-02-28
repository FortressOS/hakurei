{
  lib,
  pkgs,
  config,
  ...
}:
{
  users.users = {
    alice = {
      isNormalUser = true;
      description = "Alice Foobar";
      password = "foobar";
      uid = 1000;
    };
    untrusted = {
      isNormalUser = true;
      description = "Untrusted user";
      password = "foobar";
      uid = 1001;

      # For deny unmapped uid test:
      packages = [ config.environment.fortify.package ];
    };
  };

  home-manager.users.alice.home.stateVersion = "24.11";

  # Automatically login on tty1 as a normal user:
  services.getty.autologinUser = "alice";

  environment = {
    systemPackages = with pkgs; [
      # For glinfo and wayland-info:
      mesa-demos
      wayland-utils

      # For D-Bus tests:
      libnotify
      mako
    ];

    variables = {
      SWAYSOCK = "/tmp/sway-ipc.sock";
      WLR_RENDERER = "pixman";
    };

    # To help with OCR:
    etc."xdg/foot/foot.ini".text = lib.generators.toINI { } {
      main = {
        font = "inconsolata:size=14";
      };
      colors = rec {
        foreground = "000000";
        background = "ffffff";
        regular2 = foreground;
      };
    };
  };

  fonts.packages = [ pkgs.inconsolata ];

  # Automatically configure and start Sway when logging in on tty1:
  programs.bash.loginShellInit = ''
    if [ "$(tty)" = "/dev/tty1" ]; then
      set -e

      mkdir -p ~/.config/sway
      (sed s/Mod4/Mod1/ /etc/sway/config &&
      echo 'output * bg ${pkgs.nixos-artwork.wallpapers.simple-light-gray.gnomeFilePath} fill' &&
      echo 'output Virtual-1 res 1680x1050') > ~/.config/sway/config

      sway --validate
      systemd-cat --identifier=sway sway && touch /tmp/sway-exit-ok
    fi
  '';

  programs.sway.enable = true;

  # For PulseAudio tests:
  security.rtkit.enable = true;
  services.pipewire = {
    enable = true;
    alsa.enable = true;
    alsa.support32Bit = true;
    pulse.enable = true;
    jack.enable = true;
  };

  virtualisation.qemu.options = [
    # Need to switch to a different GPU driver than the default one (-vga std) so that Sway can launch:
    "-vga none -device virtio-gpu-pci"

    # Increase Go test compiler performance:
    "-smp 8"
  ];

  environment.fortify = {
    enable = true;
    stateDir = "/var/lib/fortify";
    users.alice = 0;

    home-manager = _: _: { home.stateVersion = "23.05"; };

    apps = [
      {
        name = "check-sandbox";
        verbose = true;
        share = pkgs.foot;
        packages = [ ];
        command = "${pkgs.callPackage ./sandbox {
          inherit (config.environment.fortify.package) version;
        }}";
        extraPaths = [
          {
            src = "/proc/mounts";
            dst = "/.fortify/host-mounts";
          }
        ];
      }
      {
        name = "ne-foot";
        verbose = true;
        share = pkgs.foot;
        packages = [ pkgs.foot ];
        command = "foot";
        capability = {
          dbus = false;
          pulse = false;
        };
      }
      {
        name = "pa-foot";
        verbose = true;
        share = pkgs.foot;
        packages = [ pkgs.foot ];
        command = "foot";
        capability.dbus = false;
      }
      {
        name = "x11-alacritty";
        verbose = true;
        share = pkgs.alacritty;
        packages = [ pkgs.alacritty ];
        command = "alacritty";
        capability = {
          wayland = false;
          x11 = true;
          dbus = false;
          pulse = false;
        };
      }
      {
        name = "da-foot";
        verbose = true;
        insecureWayland = true;
        share = pkgs.foot;
        packages = [ pkgs.foot ];
        command = "foot";
        capability = {
          dbus = false;
          pulse = false;
        };
      }
      {
        name = "strace-failure";
        verbose = true;
        share = pkgs.strace;
        command = "strace true";
        capability = {
          wayland = false;
          x11 = false;
          dbus = false;
          pulse = false;
        };
      }
    ];
  };
}
