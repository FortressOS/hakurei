<p align="center">
  <a href="https://git.gensokyo.uk/security/hakurei">
    <picture>
      <img src="https://basement.gensokyo.uk/images/yukari1.png" width="200px" alt="Yukari">
    </picture>
  </a>
</p>

<p align="center">
  <a href="https://pkg.go.dev/hakurei.app"><img src="https://pkg.go.dev/badge/hakurei.app.svg" alt="Go Reference" /></a>
  <a href="https://goreportcard.com/report/hakurei.app"><img src="https://goreportcard.com/badge/hakurei.app" alt="Go Report Card" /></a>
</p>

Hakurei is a tool for running sandboxed graphical applications as dedicated subordinate users on the Linux kernel.
It also implements [planterette (WIP)](cmd/planterette), a self-contained Android-like package manager with modern security features.

## NixOS Module usage

The NixOS module currently requires home-manager to configure subordinate users. Full module documentation can be found [here](options.md).

To use the module, import it into your configuration with

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";

    hakurei = {
      url = "git+https://git.gensokyo.uk/security/hakurei";

      # Optional but recommended to limit the size of your system closure.
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, hakurei, ... }:
  {
    nixosConfigurations.hakurei = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        hakurei.nixosModules.hakurei
      ];
    };
  };
}
```

This adds the `environment.hakurei` option:

```nix
{ pkgs, ... }:

{
  environment.hakurei = {
    enable = true;
    stateDir = "/var/lib/hakurei";
    users = {
      alice = 0;
      nixos = 10;
    };

    commonPaths = [
      {
        src = "/sdcard";
        write = true;
      }
    ];

    extraHomeConfig = {
      home.stateVersion = "23.05";
    };

    apps = {
      "org.chromium.Chromium" = {
        name = "chromium";
        identity = 1;
        packages = [ pkgs.chromium ];
        userns = true;
        mapRealUid = true;
        dbus = {
          system = {
            filter = true;
            talk = [
              "org.bluez"
              "org.freedesktop.Avahi"
              "org.freedesktop.UPower"
            ];
          };
          session =
            f:
            f {
              talk = [
                "org.freedesktop.FileManager1"
                "org.freedesktop.Notifications"
                "org.freedesktop.ScreenSaver"
                "org.freedesktop.secrets"
                "org.kde.kwalletd5"
                "org.kde.kwalletd6"
              ];
              own = [
                "org.chromium.Chromium.*"
                "org.mpris.MediaPlayer2.org.chromium.Chromium.*"
                "org.mpris.MediaPlayer2.chromium.*"
              ];
              call = { };
              broadcast = { };
            };
        };
      };

      "org.claws_mail.Claws-Mail" = {
        name = "claws-mail";
        identity = 2;
        packages = [ pkgs.claws-mail ];
        gpu = false;
        capability.pulse = false;
      };

      "org.weechat" = {
        name = "weechat";
        identity = 3;
        shareUid = true;
        packages = [ pkgs.weechat ];
        capability = {
          wayland = false;
          x11 = false;
          dbus = true;
          pulse = false;
        };
      };

      "dev.vencord.Vesktop" = {
        name = "discord";
        identity = 3;
        shareUid = true;
        packages = [ pkgs.vesktop ];
        share = pkgs.vesktop;
        command = "vesktop --ozone-platform-hint=wayland";
        userns = true;
        mapRealUid = true;
        capability.x11 = true;
        dbus = {
          session =
            f:
            f {
              talk = [ "org.kde.StatusNotifierWatcher" ];
              own = [ ];
              call = { };
              broadcast = { };
            };
          system.filter = true;
        };
      };

      "io.looking-glass" = {
        name = "looking-glass-client";
        identity = 4;
        useCommonPaths = false;
        groups = [ "plugdev" ];
        extraPaths = [
          {
            src = "/dev/shm/looking-glass";
            write = true;
          }
        ];
        extraConfig = {
          programs.looking-glass-client.enable = true;
        };
      };
    };
  };
}
```
