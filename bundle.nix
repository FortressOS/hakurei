{
  nixpkgsFor,
  system,
  nixpkgs,
  home-manager,
}:

{
  lib,
  writeScript,
  runtimeShell,
  writeText,
  vmTools,
  runCommand,
  fetchFromGitHub,

  nix,

  name ? throw "name is required",
  version ? throw "version is required",
  pname ? "${name}-${version}",
  modules ? [ ],
  script ? ''
    exec "$SHELL" "$@"
  '',

  id ? name,
  app_id ? throw "app_id is required",
  groups ? [ ],
  userns ? false,
  net ? true,
  dev ? false,
  no_new_session ? false,
  map_real_uid ? false,
  direct_wayland ? false,
  system_bus ? null,
  session_bus ? null,

  allow_wayland ? true,
  allow_x11 ? false,
  allow_dbus ? true,
  allow_pulse ? true,
  gpu ? allow_wayland || allow_x11,
}:

let
  inherit (lib) optionals;

  homeManagerConfiguration = home-manager.lib.homeManagerConfiguration {
    pkgs = nixpkgsFor.${system};
    modules = modules ++ [
      {
        home = {
          username = "fortify";
          homeDirectory = "/data/data/${id}";
          stateVersion = "22.11";
        };
      }
    ];
  };

  launcher = writeScript "fortify-${pname}" ''
    #!${runtimeShell} -el
    ${script}
  '';

  extraNixOSConfig =
    { pkgs, ... }:
    {
      environment = {
        etc.nixpkgs.source = nixpkgs.outPath;
        systemPackages = [ pkgs.nix ];
      };
    };
  nixos = nixpkgs.lib.nixosSystem {
    inherit system;
    modules = [
      extraNixOSConfig
      { nix.settings.experimental-features = [ "flakes" ]; }
      { nix.settings.experimental-features = [ "nix-command" ]; }
      { boot.isContainer = true; }
      { system.stateVersion = "22.11"; }
    ];
  };

  etc = vmTools.runInLinuxVM (
    runCommand "etc" { } ''
      mkdir -p /etc
      ${nixos.config.system.build.etcActivationCommands}

      # remove unused files
      rm -rf /etc/sudoers

      mkdir -p $out
      tar -C /etc -cf "$out/etc.tar" .
    ''
  );

  extendSessionDefault = id: ext: {
    filter = true;

    talk = [ "org.freedesktop.Notifications" ] ++ ext.talk;
    own =
      (optionals (id != null) [
        "${id}.*"
        "org.mpris.MediaPlayer2.${id}.*"
      ])
      ++ ext.own;

    inherit (ext) call broadcast;
  };

  nixGL = fetchFromGitHub {
    owner = "nix-community";
    repo = "nixGL";
    rev = "310f8e49a149e4c9ea52f1adf70cdc768ec53f8a";
    hash = "sha256-lnzZQYG0+EXl/6NkGpyIz+FEOc/DSEG57AP1VsdeNrM=";
  };

  info = builtins.toJSON {
    inherit
      name
      version
      id
      app_id
      launcher
      groups
      userns
      net
      dev
      no_new_session
      map_real_uid
      direct_wayland
      system_bus
      gpu
      ;

    session_bus =
      if session_bus != null then
        (session_bus (extendSessionDefault id))
      else
        (extendSessionDefault id {
          talk = [ ];
          own = [ ];
          call = { };
          broadcast = { };
        });

    enablements =
      (if allow_wayland then 1 else 0)
      + (if allow_x11 then 2 else 0)
      + (if allow_dbus then 4 else 0)
      + (if allow_pulse then 8 else 0);

    nix_gl = if gpu then nixGL else null;
    current_system = nixos.config.system.build.toplevel;
    activation_package = homeManagerConfiguration.activationPackage;
  };
in

writeScript "fortify-${pname}-bundle-prelude" ''
  #!${runtimeShell} -el
  OUT="$(mktemp -d)"
  TAR="$(mktemp -u)"
  set -x

  nix copy --no-check-sigs --to "$OUT" "${nix}" "${nixos.config.system.build.toplevel}"
  nix store --store "$OUT" optimise
  chmod -R +r "$OUT/nix/var"
  nix copy --no-check-sigs --to "file://$OUT/res?compression=zstd&compression-level=19&parallel-compression=true" \
    "${homeManagerConfiguration.activationPackage}" \
    "${launcher}" ${if gpu then nixGL else ""}
  mkdir -p "$OUT/etc"
  tar -C "$OUT/etc" -xf "${etc}/etc.tar"
  cp "${writeText "bundle.json" info}" "$OUT/bundle.json"

  # creating an intermediate file improves zstd performance
  tar -C "$OUT" -cf "$TAR" .
  chmod +w -R "$OUT" && rm -rf "$OUT"

  zstd -T0 -19 -fo "${pname}.pkg" "$TAR"
  rm "$TAR"
''
