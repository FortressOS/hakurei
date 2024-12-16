{
  lib,
  pkgs,
  config,
  ...
}:

let
  inherit (lib)
    mkIf
    mkDefault
    mapAttrs
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
  imports = [ ./options.nix ];

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

      userdb.source = pkgs.runCommand "fortify-userdb" { } ''
        ${cfg.package}/libexec/fuserdb -o $out ${
          foldlAttrs (
            acc: username: fid:
            acc + " ${username}:${toString fid}"
          ) "-s /run/current-system/sw/bin/nologin -d ${cfg.stateDir}" cfg.users
        }
      '';
    };

    systemd.services.nix-daemon.unitConfig.RequiresMountsFor = [ "/etc/userdb" ];

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
                      home = "${cfg.stateDir}/u${toString fid}/a${toString aid}";
                      sandbox = {
                        inherit (app)
                          userns
                          net
                          dev
                          env
                          ;
                        map_real_uid = app.mapRealUid;
                        no_new_session = app.tty;
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
