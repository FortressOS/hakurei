packages:
{
  lib,
  pkgs,
  config,
  ...
}:

let
  inherit (lib)
    lists
    attrsets
    mkMerge
    mkIf
    mapAttrs
    foldlAttrs
    optional
    optionals
    ;

  cfg = config.environment.hakurei;

  # userid*userOffset + appStart + appid
  getsubuid = userid: appid: userid * 100000 + 10000 + appid;
  getsubname = userid: appid: "u${toString userid}_a${toString appid}";
  getsubhome = userid: appid: "${cfg.stateDir}/u${toString userid}/a${toString appid}";
in

{
  imports = [ (import ./options.nix packages) ];

  config = mkIf cfg.enable {
    assertions = [
      (
        let
          conflictingApps = foldlAttrs (
            acc: id: app:
            (
              acc
              ++ foldlAttrs (
                acc': id': app':
                if id == id' || app.shareUid && app'.shareUid || app.identity != app'.identity then acc' else acc' ++ [ id ]
              ) [ ] cfg.apps
            )
          ) [ ] cfg.apps;
        in
        {
          assertion = (lists.length conflictingApps) == 0;
          message = "the following hakurei apps have conflicting identities: " + (builtins.concatStringsSep ", " conflictingApps);
        }
      )
    ];

    security.wrappers.hsu = {
      source = "${cfg.hsuPackage}/bin/hsu";
      setuid = true;
      owner = "root";
      group = "root";
    };

    environment.etc.hsurc = {
      mode = "0400";
      text = foldlAttrs (
        acc: username: fid:
        "${toString config.users.users.${username}.uid} ${toString fid}\n" + acc
      ) "" cfg.users;
    };

    home-manager =
      let
        privPackages = mapAttrs (username: userid: {
          home.packages = foldlAttrs (
            acc: id: app:
            [
              (
                let
                  extendDBusDefault = id: ext: {
                    filter = true;

                    talk = [ "org.freedesktop.Notifications" ] ++ ext.talk;
                    own = [
                      "${id}.*"
                      "org.mpris.MediaPlayer2.${id}.*"
                    ]
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
                      session_bus = if app.dbus.session != null then (app.dbus.session (extendDBusDefault id)) else (extendDBusDefault id default);
                      system_bus = app.dbus.system;
                    };
                  command = if app.command == null then app.name else app.command;
                  script = if app.script == null then ("exec " + command + " $@") else app.script;
                  isGraphical = if app.gpu != null then app.gpu else app.enablements.wayland || app.enablements.x11;

                  conf = {
                    inherit id;
                    inherit (app) identity groups enablements;
                    inherit (dbusConfig) session_bus system_bus;
                    direct_wayland = app.insecureWayland;

                    container = {
                      inherit (app)
                        wait_delay
                        devel
                        userns
                        device
                        tty
                        multiarch
                        env
                        ;
                      map_real_uid = app.mapRealUid;
                      host_net = app.hostNet;
                      host_abstract = app.hostAbstract;
                      share_runtime = app.shareRuntime;
                      share_tmpdir = app.shareTmpdir;

                      filesystem =
                        let
                          bind = src: {
                            type = "bind";
                            inherit src;
                          };
                          optBind = src: {
                            type = "bind";
                            inherit src;
                            optional = true;
                          };
                          optDevBind = src: {
                            type = "bind";
                            inherit src;
                            dev = true;
                            optional = true;
                          };
                        in
                        [
                          (bind "/bin")
                          (bind "/usr/bin")
                          (bind "/nix/store")
                          (optBind "/sys/block")
                          (optBind "/sys/bus")
                          (optBind "/sys/class")
                          (optBind "/sys/dev")
                          (optBind "/sys/devices")
                        ]
                        ++ optionals app.nix [
                          (bind "/nix/var")
                        ]
                        ++ optionals isGraphical [
                          (optDevBind "/dev/dri")
                          (optDevBind "/dev/nvidiactl")
                          (optDevBind "/dev/nvidia-modeset")
                          (optDevBind "/dev/nvidia-uvm")
                          (optDevBind "/dev/nvidia-uvm-tools")
                          (optDevBind "/dev/nvidia0")
                        ]
                        ++ optionals app.useCommonPaths cfg.commonPaths
                        ++ app.extraPaths
                        ++ [
                          {
                            type = "bind";
                            dst = "/etc/";
                            src = "/etc/";
                            special = true;
                          }
                          {
                            type = "link";
                            dst = "/run/current-system";
                            linkname = "/run/current-system";
                            dereference = true;
                          }
                        ]
                        ++ optionals (isGraphical && config.hardware.graphics.enable) (
                          [
                            {
                              type = "link";
                              dst = "/run/opengl-driver";
                              linkname = config.systemd.tmpfiles.settings.graphics-driver."/run/opengl-driver"."L+".argument;
                            }
                          ]
                          ++ optionals (app.multiarch && config.hardware.graphics.enable32Bit) [
                            {
                              type = "link";
                              dst = "/run/opengl-driver-32";
                              linkname = config.systemd.tmpfiles.settings.graphics-driver."/run/opengl-driver-32"."L+".argument;
                            }
                          ]
                        )
                        ++ [
                          {
                            type = "bind";
                            src = getsubhome userid app.identity;
                            write = true;
                            ensure = true;
                          }
                        ];

                      username = getsubname userid app.identity;
                      inherit (cfg) shell;
                      home = getsubhome userid app.identity;

                      path =
                        if app.path == null then
                          pkgs.writeScript "${app.name}-start" ''
                            #!${pkgs.zsh}${pkgs.zsh.shellPath}
                            ${script}
                          ''
                        else
                          app.path;
                      args = if app.args == null then [ "${app.name}-start" ] else app.args;
                    };
                  };

                  checkedConfig =
                    name: value:
                    let
                      file = pkgs.writeText name (builtins.toJSON value);
                    in
                    pkgs.runCommand "checked-${name}" { nativeBuildInputs = [ cfg.package ]; } ''
                      ln -vs ${file} "$out"
                      hakurei show --no-store ${file}
                    '';
                in
                pkgs.writeShellScriptBin app.name ''
                  exec hakurei${if app.verbose then " -v" else ""} app ${checkedConfig "hakurei-app-${app.name}.json" conf} $@
                ''
              )
            ]
            ++ (
              let
                pkg = if app.share != null then app.share else pkgs.${app.name};
                copy = source: "[ -d '${source}' ] && cp -Lrv '${source}' $out/share || true";
              in
              optional (app.enablements.wayland || app.enablements.x11) (
                pkgs.runCommand "${app.name}-share" { } ''
                  mkdir -p $out/share
                  ${copy "${pkg}/share/applications"}
                  ${copy "${pkg}/share/pixmaps"}
                  ${copy "${pkg}/share/icons"}
                  ${copy "${pkg}/share/man"}

                  if test -d "$out/share/applications"; then
                    substituteInPlace $out/share/applications/* \
                      --replace-warn '${pkg}/bin/' "" \
                      --replace-warn '${pkg}/libexec/' ""
                  fi
                ''
              )
            )
            ++ acc
          ) [ cfg.package ] cfg.apps;
        }) cfg.users;
      in
      {
        useUserPackages = false; # prevent users.users entries from being added

        users =
          mkMerge
            (foldlAttrs
              (
                acc: _: fid:
                foldlAttrs
                  (
                    acc: _: app:
                    (
                      let
                        key = getsubname fid app.identity;
                      in
                      {
                        usernames = acc.usernames // {
                          ${key} = true;
                        };
                        merge = acc.merge ++ [
                          {
                            ${key} = mkMerge (
                              [
                                app.extraConfig
                                { home.packages = app.packages; }
                              ]
                              ++ lib.optional (!attrsets.hasAttrByPath [ key ] acc.usernames) cfg.extraHomeConfig
                            );
                          }
                        ];
                      }
                    )
                  )
                  {
                    inherit (acc) usernames;
                    merge = acc.merge ++ [ { ${getsubname fid 0} = cfg.extraHomeConfig; } ];
                  }
                  cfg.apps
              )
              {
                usernames = { };
                merge = [ privPackages ];
              }
              cfg.users
            ).merge;
      };

    users =
      let
        getuser = userid: appid: {
          isSystemUser = true;
          createHome = true;
          description = "Hakurei subordinate user ${toString appid} (u${toString userid})";
          group = getsubname userid appid;
          home = getsubhome userid appid;
          uid = getsubuid userid appid;
        };
        getgroup = userid: appid: { gid = getsubuid userid appid; };
      in
      {
        users = mkMerge (
          foldlAttrs (
            acc: _: fid:
            acc
            ++ foldlAttrs (
              acc': _: app:
              acc' ++ [ { ${getsubname fid app.identity} = getuser fid app.identity; } ]
            ) [ { ${getsubname fid 0} = getuser fid 0; } ] cfg.apps
          ) [ ] cfg.users
        );

        groups = mkMerge (
          foldlAttrs (
            acc: _: fid:
            acc
            ++ foldlAttrs (
              acc': _: app:
              acc' ++ [ { ${getsubname fid app.identity} = getgroup fid app.identity; } ]
            ) [ { ${getsubname fid 0} = getgroup fid 0; } ] cfg.apps
          ) [ ] cfg.users
        );
      };
  };
}
