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
    mapAttrs
    mapAttrsToList
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

      target = mkOption {
        default = { };
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
          attrsOf (submodule {
            options = {
              packages = mkOption {
                type = listOf package;
                default = [ ];
                description = ''
                  List of extra packages to install via home-manager.
                '';
              };

              launchers = mkOption {
                type = attrsOf (submodule {
                  options = {
                    id = mkOption {
                      type = nullOr str;
                      default = null;
                      description = ''
                        Freedesktop application ID.
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
                        Setting this to null will default command to wrapper name.
                        Has no effect when script is set.
                      '';
                    };

                    method = mkOption {
                      type = enum [
                        "simple"
                        "sudo"
                        "systemd"
                      ];
                      default = "systemd";
                      description = ''
                        Launch method for the sandboxed program.
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

                    useRealUid = mkEnableOption ''
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
                default = { };
              };

              persistence = mkOption {
                type = submodule {
                  options = {
                    directories = mkOption {
                      type = listOf anything;
                      default = [ ];
                    };

                    files = mkOption {
                      type = listOf anything;
                      default = [ ];
                    };
                  };
                };
                description = ''
                  Per-user state passed to github:nix-community/impermanence.
                '';
              };

              extraConfig = mkOption {
                type = anything;
                default = { };
                description = "Extra home-manager configuration.";
              };
            };
          });
      };

      package = mkOption {
        type = types.package;
        default = pkgs.callPackage ./package.nix { };
        description = "Package providing fortify.";
      };

      user = mkOption {
        type = types.str;
        description = "Privileged user account.";
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
    environment.persistence.${cfg.stateDir}.users = mapAttrs (_: target: target.persistence) cfg.target;

    home-manager.users =
      mapAttrs (_: target: target.extraConfig // { home.packages = target.packages; }) cfg.target
      // {
        ${cfg.user}.home.packages =
          let
            wrap =
              user: launchers:
              mapAttrsToList (
                name: launcher:
                with launcher.capability;
                let
                  extendDBusDefault = id: ext: {
                    filter = true;

                    talk = [ "org.freedesktop.Notifications" ] ++ ext.talk;
                    own =
                      (optionals (launcher.id != null) [
                        "${id}.*"
                        "org.mpris.MediaPlayer2.${id}.*"
                      ])
                      ++ ext.own;
                    call = {
                      "org.freedesktop.portal.*" = "*";
                    } // ext.call;
                    broadcast = {
                      "org.freedesktop.portal.*" = "@/org/freedesktop/portal/*";
                    } // ext.broadcast;
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
                        if launcher.dbus.session != null then
                          (launcher.dbus.session (extendDBusDefault launcher.id))
                        else
                          (extendDBusDefault launcher.id default);
                      system_bus = launcher.dbus.system;
                    };
                  command = if launcher.command == null then name else launcher.command;
                  script = if launcher.script == null then ("exec " + command + " $@") else launcher.script;
                  enablements =
                    (if wayland then 1 else 0)
                    + (if x11 then 2 else 0)
                    + (if dbus then 4 else 0)
                    + (if pulse then 8 else 0);
                  conf = {
                    inherit (launcher) id method;
                    inherit user;
                    command = [
                      (pkgs.writeScript "${name}-start" ''
                        #!${pkgs.zsh}${pkgs.zsh.shellPath}
                        ${script}
                      '')
                    ];
                    confinement = {
                      sandbox = {
                        inherit (launcher)
                          userns
                          net
                          dev
                          env
                          ;
                        use_real_uid = launcher.useRealUid;
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
                            {
                              src = "/home/${user}";
                              write = true;
                              require = true;
                            }
                          ]
                          ++ optionals launcher.nix [
                            { src = "/nix/var"; }
                            { src = "/var/db/nix-channels"; }
                          ]
                          ++ optionals (if launcher.gpu != null then launcher.gpu else wayland || x11) [
                            { src = "/run/opengl-driver"; }
                            {
                              src = "/dev/dri";
                              dev = true;
                            }
                          ]
                          ++ launcher.extraPaths;
                        auto_etc = true;
                        override = [ "/var/run/nscd" ];
                      };
                      inherit enablements;
                      inherit (dbusConfig) session_bus system_bus;
                    };
                  };
                in
                pkgs.writeShellScriptBin name (
                  if launcher.method == "simple" then
                    ''
                      exec sudo -u ${user} -i ${command} $@
                    ''
                  else
                    ''
                      exec fortify app ${pkgs.writeText "fortify-${name}.json" (builtins.toJSON conf)} $@
                    ''
                )
              ) launchers;
          in
          foldlAttrs (
            acc: user: target:
            acc
            ++ (foldlAttrs (
              shares: name: launcher:
              let
                pkg = if launcher.share != null then launcher.share else pkgs.${name};
                link = source: "[ -d '${source}' ] && ln -sv '${source}' $out/share || true";
              in
              shares
              ++
                optional (launcher.method != "simple" && (launcher.capability.wayland || launcher.capability.x11))
                  (
                    pkgs.runCommand "${name}-share" { } ''
                      mkdir -p $out/share
                      ${link "${pkg}/share/applications"}
                      ${link "${pkg}/share/icons"}
                      ${link "${pkg}/share/man"}
                    ''
                  )
            ) (wrap user target.launchers) target.launchers)
          ) [ cfg.package ] cfg.target;
      };

    security.polkit.extraConfig =
      let
        allowList = builtins.toJSON (mapAttrsToList (name: _: name) cfg.target);
      in
      ''
        polkit.addRule(function(action, subject) {
          if (action.id == "org.freedesktop.machine1.host-shell" &&
            ${allowList}.indexOf(action.lookup("user")) > -1 &&
            subject.user == "${cfg.user}") {
                return polkit.Result.YES;
          }
        });
      '';
  };
}
