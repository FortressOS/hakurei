packages:
{ lib, pkgs, ... }:

let
  inherit (lib) types mkOption mkEnableOption;

  mountPoint =
    let
      inherit (types)
        str
        submodule
        nullOr
        listOf
        ;
    in
    listOf (submodule {
      options = {
        dst = mkOption {
          type = nullOr str;
          default = null;
          description = ''
            Mount point in container, same as src if null.
          '';
        };

        src = mkOption {
          type = str;
          description = ''
            Host filesystem path to make available to the container.
          '';
        };

        write = mkEnableOption "mounting path as writable";
        dev = mkEnableOption "use of device files";
        require = mkEnableOption "start failure if the bind mount cannot be established for any reason";
      };
    });
in

{
  options = {
    environment.fortify = {
      enable = mkEnableOption "fortify";

      package = mkOption {
        type = types.package;
        default = packages.${pkgs.system}.fortify;
        description = "The fortify package to use.";
      };

      fsuPackage = mkOption {
        type = types.package;
        default = packages.${pkgs.system}.fsu;
        description = "The fsu package to use.";
      };

      users = mkOption {
        type =
          let
            inherit (types) attrsOf ints;
          in
          attrsOf (ints.between 0 99);
        description = ''
          Users allowed to spawn fortify apps and their corresponding fortify fid.
        '';
      };

      extraHomeConfig = mkOption {
        type = types.anything;
        description = ''
          Extra home-manager configuration to merge with all target users.
        '';
      };

      apps = mkOption {
        type =
          let
            inherit (types)
              str
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
                  Name of the app's launcher script.
                '';
              };

              verbose = mkEnableOption "launchers with verbose output";

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
                description = ''
                  Extra home-manager configuration.
                '';
              };

              path = mkOption {
                type = nullOr str;
                default = null;
                description = ''
                  Custom executable path.
                  Setting this to null will default to the start script.
                '';
              };

              args = mkOption {
                type = nullOr (listOf str);
                default = null;
                description = ''
                  Custom args.
                  Setting this to null will default to script name.
                '';
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
                  Setting this to null will default command to launcher name.
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

              devel = mkEnableOption "debugging-related kernel interfaces";
              userns = mkEnableOption "user namespace creation";
              tty = mkEnableOption "access to the controlling terminal";
              multiarch = mkEnableOption "multiarch kernel-level support";

              net = mkEnableOption "network access" // {
                default = true;
              };

              nix = mkEnableOption "nix daemon access";
              mapRealUid = mkEnableOption "mapping to priv-user uid";
              device = mkEnableOption "access to all devices";
              insecureWayland = mkEnableOption "direct access to the Wayland socket";

              gpu = mkOption {
                type = nullOr bool;
                default = null;
                description = ''
                  Target process GPU and driver access.
                  Setting this to null will enable GPU whenever X or Wayland is enabled.
                '';
              };

              useCommonPaths = mkEnableOption "common extra paths" // {
                default = true;
              };

              extraPaths = mkOption {
                type = mountPoint;
                default = [ ];
                description = ''
                  Extra paths to make available to the container.
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
        description = ''
          Declaratively configured fortify apps.
        '';
      };

      commonPaths = mkOption {
        type = mountPoint;
        default = [ ];
        description = ''
          Common extra paths to make available to the container.
        '';
      };

      stateDir = mkOption {
        type = types.str;
        description = ''
          The state directory where app home directories are stored.
        '';
      };
    };
  };
}
