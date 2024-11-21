Fortify
=======

[![Go Reference](https://pkg.go.dev/badge/git.ophivana.moe/security/fortify.svg)](https://pkg.go.dev/git.ophivana.moe/security/fortify)
[![Go Report Card](https://goreportcard.com/badge/git.ophivana.moe/security/fortify)](https://goreportcard.com/report/git.ophivana.moe/security/fortify)

Lets you run graphical applications as another user in a confined environment with a nice NixOS
module to configure target users and provide launchers and desktop files for your privileged user.

Why would you want this?

- It protects the desktop environment from applications.

- It protects applications from each other.

- It provides UID isolation on top of the standard application sandbox.

There are a few different things to set up for this to work:

- A set of users, each for a group of applications that should be allowed access to each other

- A tool to switch users, currently sudo and machinectl are supported.

- If you are running NixOS, the module in this repository can take care of launchers and desktop files in the privileged
  user's environment, as well as packages and extra home-manager configuration for target users.

If you have a flakes-enabled nix environment, you can try out the tool by running:

```shell
nix run git+https://git.ophivana.moe/security/fortify -- -h
```

## Module usage

The NixOS module currently requires home-manager to function correctly.

Full module documentation can be found [here](options.md).

To use the module, import it into your configuration with

```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";

    fortify = {
      url = "git+https://git.ophivana.moe/security/fortify";

      # Optional but recommended to limit the size of your system closure.
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, fortify, ... }:
  {
    nixosConfigurations.fortify = nixpkgs.lib.nixosSystem {
      system = "x86_64-linux";
      modules = [
        fortify.nixosModules.fortify
      ];
    };
  };
}
```

This adds the `environment.fortify` option:

```nix
{ pkgs, ... }:

{
  environment.fortify = {
    enable = true;
    stateDir = "/var/lib/persist/module/fortify";
    users = {
      alice = 0;
      nixos = 10;
    };

    apps = [
      {
        name = "chromium";
        id = "org.chromium.Chromium";
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
      }
      {
        name = "claws-mail";
        id = "org.claws_mail.Claws-Mail";
        packages = [ pkgs.claws-mail ];
        gpu = false;
        capability.pulse = false;
      }
      {
        name = "weechat";
        packages = [ pkgs.weechat ];
        capability = {
          wayland = false;
          x11 = false;
          dbus = true;
          pulse = false;
        };
      }
      {
        name = "discord";
        id = "dev.vencord.Vesktop";
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
      }
      {
        name = "looking-glass-client";
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
      }
    ];
  };
}
```
