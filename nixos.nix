{
  lib,
  pkgs,
  config,
  ...
}:

let
  inherit (lib)
    types
    mkOption
    mkEnableOption
    mkIf
    mkDefault
    mapAttrs
    mapAttrsToList
    mergeAttrsList
    imap1
    foldr
    foldlAttrs
    optional
    optionals
    ;

  cfg = config.environment.fortify;
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

  config = mkIf cfg.enable {
    security.wrappers.fsu = {
      source = "${cfg.package}/libexec/fsu";
      setuid = true;
      owner = "root";
      setgid = true;
      group = "root";
    };

    environment.etc = {
      fsurc = {
        mode = "0400";
        text = foldlAttrs (
          acc: username: fid:
          "${toString config.users.users.${username}.uid} ${toString fid}\n" + acc
        ) "" cfg.users;
      };

      userdb.source = pkgs.runCommand "generate-userdb" { } ''
        ${cfg.package}/libexec/fuserdb -o $out ${
          foldlAttrs (
            acc: username: fid:
            acc + " ${username}:${toString fid}"
          ) "-s /run/current-system/sw/bin/nologin -d ${cfg.stateDir}" cfg.users
        }
      '';
    };

    services.userdbd.enable = mkDefault true;

    home-manager =
      let
        privPackages = mapAttrs (username: fid: {
          home.packages =
            let
              # aid 0 is reserved
              wrappers = imap1 (
                aid: app:
                let
                  extendDBusDefault = id: ext: {
                    filter = true;

                    talk = [ "org.freedesktop.Notifications" ] ++ ext.talk;
                    own =
                      (optionals (app.id != null) [
                        "${id}.*"
                        "org.mpris.MediaPlayer2.${id}.*"
                      ])
                      ++ ext.own;

                    inherit (ext) call broadcast;
                  };
                  dbusConfig =
                    let
                      default = {
                        talk = [ ];
                        own = [ ];
                        call = { };
                        broadcast = { };
                      };
                    in
                    {
                      session_bus =
                        if app.dbus.session != null then
                          (app.dbus.session (extendDBusDefault app.id))
                        else
                          (extendDBusDefault app.id default);
                      system_bus = app.dbus.system;
                    };
                  command = if app.command == null then app.name else app.command;
                  script = if app.script == null then ("exec " + command + " $@") else app.script;
                  enablements =
                    with app.capability;
                    (if wayland then 1 else 0)
                    + (if x11 then 2 else 0)
                    + (if dbus then 4 else 0)
                    + (if pulse then 8 else 0);
                  conf = {
                    inherit (app) id;
                    command = [
                      (pkgs.writeScript "${app.name}-start" ''
                        #!${pkgs.zsh}${pkgs.zsh.shellPath}
                        ${script}
                      '')
                    ];
                    confinement = {
                      app_id = aid;
                      inherit (app) groups;
                      username = "u${toString fid}_a${toString aid}";
                      home = "${cfg.stateDir}/${toString fid}/${toString aid}";
                      sandbox = {
                        inherit (app)
                          userns
                          net
                          dev
                          env
                          ;
                        map_real_uid = app.mapRealUid;
                        filesystem =
                          [
                            { src = "/bin"; }
                            { src = "/usr/bin"; }
                            { src = "/nix/store"; }
                            { src = "/run/current-system"; }
                            {
                              src = "/sys/block";
                              require = false;
                            }
                            {
                              src = "/sys/bus";
                              require = false;
                            }
                            {
                              src = "/sys/class";
                              require = false;
                            }
                            {
                              src = "/sys/dev";
                              require = false;
                            }
                            {
                              src = "/sys/devices";
                              require = false;
                            }
                          ]
                          ++ optionals app.nix [
                            { src = "/nix/var"; }
                            { src = "/var/db/nix-channels"; }
                          ]
                          ++ optionals (if app.gpu != null then app.gpu else app.capability.wayland || app.capability.x11) [
                            { src = "/run/opengl-driver"; }
                            {
                              src = "/dev/dri";
                              dev = true;
                            }
                          ]
                          ++ app.extraPaths;
                        auto_etc = true;
                        override = [ "/var/run/nscd" ];
                      };
                      inherit enablements;
                      inherit (dbusConfig) session_bus system_bus;
                    };
                  };
                in
                pkgs.writeShellScriptBin app.name ''
                  exec fortify app ${pkgs.writeText "fortify-${app.name}.json" (builtins.toJSON conf)} $@
                ''
              ) cfg.apps;
            in
            foldr (
              app: acc:
              let
                pkg = if app.share != null then app.share else pkgs.${app.name};
                copy = source: "[ -d '${source}' ] && cp -Lrv '${source}' $out/share || true";
              in
              optional (app.capability.wayland || app.capability.x11) (
                pkgs.runCommand "${app.name}-share" { } ''
                  mkdir -p $out/share
                  ${copy "${pkg}/share/applications"}
                  ${copy "${pkg}/share/pixmaps"}
                  ${copy "${pkg}/share/icons"}
                  ${copy "${pkg}/share/man"}

                  substituteInPlace $out/share/applications/* \
                    --replace-warn '${pkg}/bin/' "" \
                    --replace-warn '${pkg}/libexec/' ""
                ''
              )
              ++ acc
            ) (wrappers ++ [ cfg.package ]) cfg.apps;
        }) cfg.users;
      in
      {
        useUserPackages = false; # prevent users.users entries from being added

        users = foldlAttrs (
          acc: _: fid:
          mergeAttrsList (
            # aid 0 is reserved
            imap1 (aid: app: {
              "u${toString fid}_a${toString aid}" = app.extraConfig // {
                home.packages = app.packages;
              };
            }) cfg.apps
          )
          // acc
        ) privPackages cfg.users;
      };
  };
}
