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
      # For D-Bus tests:
      mako
      libnotify
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
      systemd-cat --identifier=session sway && touch /tmp/sway-exit-ok
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

    extraHomeConfig = {
      home.stateVersion = "23.05";
    };

    apps = {
      "cat.gensokyo.extern.foot.noEnablements" = {
        name = "ne-foot";
        identity = 1;
        verbose = true;
        share = pkgs.foot;
        packages = with pkgs; [
          foot

          # For wayland-info:
          wayland-utils
        ];
        command = "foot";
        capability = {
          dbus = false;
          pulse = false;
        };
      };

      "cat.gensokyo.extern.foot.pulseaudio" = {
        name = "pa-foot";
        identity = 2;
        verbose = true;
        share = pkgs.foot;
        packages = [ pkgs.foot ];
        command = "foot";
        capability.dbus = false;
      };

      "cat.gensokyo.extern.Alacritty.x11" = {
        name = "x11-alacritty";
        identity = 3;
        verbose = true;
        share = pkgs.alacritty;
        packages = with pkgs; [
          # For X11 terminal emulator:
          alacritty

          # For glinfo:
          mesa-demos
        ];
        command = "alacritty";
        capability = {
          wayland = false;
          x11 = true;
          dbus = false;
          pulse = false;
        };
      };

      "cat.gensokyo.extern.foot.directWayland" = {
        name = "da-foot";
        identity = 4;
        verbose = true;
        insecureWayland = true;
        share = pkgs.foot;
        packages = with pkgs; [
          foot

          # For wayland-info:
          wayland-utils
        ];
        command = "foot";
        capability = {
          dbus = false;
          pulse = false;
        };
      };

      "cat.gensokyo.extern.strace.wantFail" = {
        name = "strace-failure";
        identity = 5;
        verbose = true;
        share = pkgs.strace;
        command = "strace true";
        capability = {
          wayland = false;
          x11 = false;
          dbus = false;
          pulse = false;
        };
      };
    };
  };
}
