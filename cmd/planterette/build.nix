{
  nixpkgsFor,
  system,
  nixpkgs,
  home-manager,
}:

{
  lib,
  stdenv,
  closureInfo,
  writeScript,
  runtimeShell,
  writeText,
  symlinkJoin,
  vmTools,
  runCommand,
  fetchFromGitHub,

  zstd,
  nix,
  sqlite,

  name ? throw "name is required",
  version ? throw "version is required",
  pname ? "${name}-${version}",
  modules ? [ ],
  nixosModules ? [ ],
  script ? ''
    exec "$SHELL" "$@"
  '',

  id ? name,
  identity ? throw "identity is required",
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
          username = "hakurei";
          homeDirectory = "/data/data/${id}";
          stateVersion = "22.11";
        };
      }
    ];
  };

  launcher = writeScript "hakurei-${pname}" ''
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

      imports = nixosModules;
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

  mesaWrappers =
    let
      isIntelX86Platform = system == "x86_64-linux";
      nixGLPackages = import (nixGL + "/default.nix") {
        pkgs = nixpkgs.legacyPackages.${system};
        enable32bits = isIntelX86Platform;
        enableIntelX86Extensions = isIntelX86Platform;
      };
    in
    symlinkJoin {
      name = "nixGL-mesa";
      paths = with nixGLPackages; [
        nixGLIntel
        nixVulkanIntel
      ];
    };

  info = builtins.toJSON {
    inherit
      name
      version
      id
      identity
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

    enablements = (if allow_wayland then 1 else 0) + (if allow_x11 then 2 else 0) + (if allow_dbus then 4 else 0) + (if allow_pulse then 8 else 0);

    mesa = if gpu then mesaWrappers else null;
    nix_gl = if gpu then nixGL else null;
    current_system = nixos.config.system.build.toplevel;
    activation_package = homeManagerConfiguration.activationPackage;
  };
in

stdenv.mkDerivation {
  name = "${pname}.pkg";
  inherit version;
  __structuredAttrs = true;

  nativeBuildInputs = [
    zstd
    nix
    sqlite
  ];

  buildCommand = ''
    NIX_ROOT="$(mktemp -d)"
    export USER="nobody"

    # create bootstrap store
    bootstrapClosureInfo="${
      closureInfo {
        rootPaths = [
          nix
          nixos.config.system.build.toplevel
        ];
      }
    }"
    echo "copying bootstrap store paths..."
    mkdir -p "$NIX_ROOT/nix/store"
    xargs -n 1 -a "$bootstrapClosureInfo/store-paths" cp -at "$NIX_ROOT/nix/store/"
    NIX_REMOTE="local?root=$NIX_ROOT" nix-store --load-db < "$bootstrapClosureInfo/registration"
    NIX_REMOTE="local?root=$NIX_ROOT" nix-store --optimise
    sqlite3 "$NIX_ROOT/nix/var/nix/db/db.sqlite" "UPDATE ValidPaths SET registrationTime = ''${SOURCE_DATE_EPOCH}"
    chmod -R +r "$NIX_ROOT/nix/var"

    # create binary cache
    closureInfo="${
      closureInfo {
        rootPaths =
          [
            homeManagerConfiguration.activationPackage
            launcher
          ]
          ++ optionals gpu [
            mesaWrappers
            nixGL
          ];
      }
    }"
    echo "copying application paths..."
    TMP_STORE="$(mktemp -d)"
    mkdir -p "$TMP_STORE/nix/store"
    xargs -n 1 -a "$closureInfo/store-paths" cp -at "$TMP_STORE/nix/store/"
    NIX_REMOTE="local?root=$TMP_STORE" nix-store --load-db < "$closureInfo/registration"
    sqlite3 "$TMP_STORE/nix/var/nix/db/db.sqlite" "UPDATE ValidPaths SET registrationTime = ''${SOURCE_DATE_EPOCH}"
    NIX_REMOTE="local?root=$TMP_STORE" nix --offline --extra-experimental-features nix-command \
        --verbose --log-format raw-with-logs \
        copy --all --no-check-sigs --to \
        "file://$NIX_ROOT/res?compression=zstd&compression-level=19&parallel-compression=true"

    # package /etc
    mkdir -p "$NIX_ROOT/etc"
    tar -C "$NIX_ROOT/etc" -xf "${etc}/etc.tar"

    # write metadata
    cp "${writeText "bundle.json" info}" "$NIX_ROOT/bundle.json"

    # create an intermediate file to improve zstd performance
    INTER="$(mktemp)"
    tar -C "$NIX_ROOT" -cf "$INTER" .
    zstd -T0 -19 -fo "$out" "$INTER"
  '';
}
