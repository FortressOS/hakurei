{ lib, pkgs, ... }:

let
  inherit (lib) types mkOption mkEnableOption;
in

{
  options = {
    environment.fortify = {
      enable = mkEnableOption "fortify";

      package = mkOption {
        type = types.package;
        default = pkgs.callPackage ./package.nix { };
        description = "Package providing fortify.";
      };

      users = mkOption {
        type =
          let
            inherit (types) attrsOf ints;
          in
          attrsOf (ints.between 0 99);
        description = ''
          Users allowed to spawn fortify apps, as well as their fortify ID value.
        '';
      };

      apps = mkOption {
        type =
          let
            inherit (types)
              str
              enum
              bool
              package
              anything
              submodule
              listOf
              attrsOf
              nullOr
              functionTo
              ;
          in
          listOf (submodule {
            options = {
              name = mkOption {
                type = str;
                description = ''
                  App name, typically command.
                '';
              };

              id = mkOption {
                type = nullOr str;
                default = null;
                description = ''
                  Freedesktop application ID.
                '';
              };

              packages = mkOption {
                type = listOf package;
                default = [ ];
                description = ''
                  List of extra packages to install via home-manager.
                '';
              };

              extraConfig = mkOption {
                type = anything;
                default = { };
                description = "Extra home-manager configuration.";
              };

              script = mkOption {
                type = nullOr str;
                default = null;
                description = ''
                  Application launch script.
                '';
              };

              command = mkOption {
                type = nullOr str;
                default = null;
                description = ''
                  Command to run as the target user.
                  Setting this to null will default command to wrapper name.
                  Has no effect when script is set.
                '';
              };

              groups = mkOption {
                type = listOf str;
                default = [ ];
                description = ''
                  List of groups to inherit from the privileged user.
                '';
              };

              dbus = {
                session = mkOption {
                  type = nullOr (functionTo anything);
                  default = null;
                  description = ''
                    D-Bus session bus custom configuration.
                    Setting this to null will enable built-in defaults.
                  '';
                };

                system = mkOption {
                  type = nullOr anything;
                  default = null;
                  description = ''
                    D-Bus system bus custom configuration.
                    Setting this to null will disable the system bus proxy.
                  '';
                };
              };

              env = mkOption {
                type = nullOr (attrsOf str);
                default = null;
                description = ''
                  Environment variables to set for the initial process in the sandbox.
                '';
              };

              nix = mkEnableOption ''
                Whether to allow nix daemon connections from within sandbox.
              '';

              userns = mkEnableOption ''
                Whether to allow userns within sandbox.
              '';

              mapRealUid = mkEnableOption ''
                Whether to map to fortify's real UID within the sandbox.
              '';

              net =
                mkEnableOption ''
                  Whether to allow network access within sandbox.
                ''
                // {
                  default = true;
                };

              gpu = mkOption {
                type = nullOr bool;
                default = null;
                description = ''
                  Target process GPU and driver access.
                  Setting this to null will enable GPU whenever X or Wayland is enabled.
                '';
              };

              dev = mkEnableOption ''
                Whether to allow access to all devices within sandbox.
              '';

              extraPaths = mkOption {
                type = listOf anything;
                default = [ ];
                description = ''
                  Extra paths to make available inside the sandbox.
                '';
              };

              capability = {
                wayland = mkOption {
                  type = bool;
                  default = true;
                  description = ''
                    Whether to share the Wayland socket.
                  '';
                };

                x11 = mkOption {
                  type = bool;
                  default = false;
                  description = ''
                    Whether to share the X11 socket and allow connection.
                  '';
                };

                dbus = mkOption {
                  type = bool;
                  default = true;
                  description = ''
                    Whether to proxy D-Bus.
                  '';
                };

                pulse = mkOption {
                  type = bool;
                  default = true;
                  description = ''
                    Whether to share the PulseAudio socket and cookie.
                  '';
                };
              };

              share = mkOption {
                type = nullOr package;
                default = null;
                description = ''
                  Package containing share files.
                  Setting this to null will default package name to wrapper name.
                '';
              };
            };
          });
        default = [ ];
        description = "Applications managed by fortify.";
      };

      stateDir = mkOption {
        type = types.str;
        description = ''
          The path to persistent storage where per-user state should be stored.
        '';
      };
    };
  };
}
