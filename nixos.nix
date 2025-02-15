{
  lib,
  pkgs,
  config,
  ...
}:

let
  inherit (lib)
    mkMerge
    mkIf
    mapAttrs
    mergeAttrsList
    imap1
    foldr
    foldlAttrs
    optional
    optionals
    ;

  cfg = config.environment.fortify;

  getsubuid = fid: aid: 1000000 + fid * 10000 + aid;
  getsubname = fid: aid: "u${toString fid}_a${toString aid}";
  getsubhome = fid: aid: "${cfg.stateDir}/u${toString fid}/a${toString aid}";
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

    environment.etc.fsurc = {
      mode = "0400";
      text = foldlAttrs (
        acc: username: fid:
        "${toString config.users.users.${username}.uid} ${toString fid}\n" + acc
      ) "" cfg.users;
    };

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
                      username = getsubname fid aid;
                      home = getsubhome fid aid;
                      sandbox = {
                        inherit (app)
                          userns
                          net
                          dev
                          env
                          ;
                        syscall = {
                          inherit (app) compat multiarch bluetooth;
                          deny_devel = !app.devel;
                        };
                        map_real_uid = app.mapRealUid;
                        no_new_session = app.tty;
                        filesystem =
                          let
                            bind = src: { inherit src; };
                            mustBind = src: {
                              inherit src;
                              require = true;
                            };
                            devBind = src: {
                              inherit src;
                              dev = true;
                            };
                          in
                          [
                            (mustBind "/bin")
                            (mustBind "/usr/bin")
                            (mustBind "/nix/store")
                            (mustBind "/run/current-system")
                            (bind "/sys/block")
                            (bind "/sys/bus")
                            (bind "/sys/class")
                            (bind "/sys/dev")
                            (bind "/sys/devices")
                          ]
                          ++ optionals app.nix [
                            (mustBind "/nix/var")
                            (bind "/var/db/nix-channels")
                          ]
                          ++ optionals (if app.gpu != null then app.gpu else app.capability.wayland || app.capability.x11) [
                            (bind "/run/opengl-driver")
                            (devBind "/dev/dri")
                            (devBind "/dev/nvidiactl")
                            (devBind "/dev/nvidia-modeset")
                            (devBind "/dev/nvidia-uvm")
                            (devBind "/dev/nvidia-uvm-tools")
                            (devBind "/dev/nvidia0")
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
                  exec fortify${
                    if app.verbose then " -v" else ""
                  } app ${pkgs.writeText "fortify-${app.name}.json" (builtins.toJSON conf)} $@
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
          mkMerge [
            (mergeAttrsList (
              # aid 0 is reserved
              imap1 (aid: app: {
                ${getsubname fid aid} = mkMerge [
                  (cfg.home-manager (getsubname fid aid) (getsubuid fid aid))
                  app.extraConfig
                  { home.packages = app.packages; }
                ];
              }) cfg.apps
            ))
            { ${getsubname fid 0} = cfg.home-manager (getsubname fid 0) (getsubuid fid 0); }
            acc
          ]
        ) privPackages cfg.users;
      };

    users =
      let
        getuser = fid: aid: {
          isSystemUser = true;
          createHome = true;
          description = "Fortify subordinate user ${toString aid} (u${toString fid})";
          group = getsubname fid aid;
          home = getsubhome fid aid;
          uid = getsubuid fid aid;
        };
        getgroup = fid: aid: { gid = getsubuid fid aid; };
      in
      {
        users = foldlAttrs (
          acc: _: fid:
          mkMerge [
            (mergeAttrsList (
              # aid 0 is reserved
              imap1 (aid: _: {
                ${getsubname fid aid} = getuser fid aid;
              }) cfg.apps
            ))
            { ${getsubname fid 0} = getuser fid 0; }
            acc
          ]
        ) { } cfg.users;

        groups = foldlAttrs (
          acc: _: fid:
          mkMerge [
            (mergeAttrsList (
              # aid 0 is reserved
              imap1 (aid: _: {
                ${getsubname fid aid} = getgroup fid aid;
              }) cfg.apps
            ))
            { ${getsubname fid 0} = getgroup fid 0; }
            acc
          ]
        ) { } cfg.users;
      };
  };
}
