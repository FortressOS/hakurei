{
  lib,
  stdenv,
  buildGoModule,
  makeBinaryWrapper,
  xdg-dbus-proxy,
  pkg-config,
  libffi,
  libseccomp,
  acl,
  wayland,
  wayland-protocols,
  wayland-scanner,
  xorg,

  # for hpkg
  zstd,
  gnutar,
  coreutils,

  # for passthru.buildInputs
  go,
  gcc,

  # for check
  util-linux,

  glibc, # for ldd
  withStatic ? stdenv.hostPlatform.isStatic,
}:

buildGoModule rec {
  pname = "hakurei";
  version = "0.2.1";

  srcFiltered = builtins.path {
    name = "${pname}-src";
    path = lib.cleanSource ./.;
    filter = path: type: !(type == "regular" && (lib.hasSuffix ".nix" path || lib.hasSuffix ".py" path)) && !(type == "directory" && lib.hasSuffix "/test" path) && !(type == "directory" && lib.hasSuffix "/cmd/hsu" path);
  };
  vendorHash = null;

  src = stdenv.mkDerivation {
    name = "${pname}-src-full";
    inherit version;
    enableParallelBuilding = true;
    src = srcFiltered;

    buildInputs = [
      wayland
      wayland-protocols
    ];

    nativeBuildInputs = [
      go
      pkg-config
      wayland-scanner
    ];

    buildPhase = "GOCACHE=$(mktemp -d) go generate ./...";
    installPhase = "cp -r . $out";
  };

  ldflags =
    lib.attrsets.foldlAttrs
      (
        ldflags: name: value:
        ldflags ++ [ "-X hakurei.app/internal.${name}=${value}" ]
      )
      (
        [ "-s -w" ]
        ++ lib.optionals withStatic [
          "-linkmode external"
          "-extldflags \"-static\""
        ]
      )
      {
        version = "v${version}";
        hmain = "${placeholder "out"}/libexec/hakurei";
        hsu = "/run/wrappers/bin/hsu";
      };

  # nix build environment does not allow acls
  env.GO_TEST_SKIP_ACL = 1;

  buildInputs = [
    libffi
    libseccomp
    acl
    wayland
  ]
  ++ (with xorg; [
    libxcb
    libXau
    libXdmcp
  ]);

  nativeBuildInputs = [
    pkg-config
    makeBinaryWrapper
  ];

  postInstall =
    let
      appPackages = [
        glibc
        xdg-dbus-proxy
      ];
    in
    ''
      install -D --target-directory=$out/share/zsh/site-functions dist/comp/*

      mkdir "$out/libexec"
      mv "$out"/bin/* "$out/libexec/"

      makeBinaryWrapper "$out/libexec/hakurei" "$out/bin/hakurei" \
        --inherit-argv0 --prefix PATH : ${lib.makeBinPath appPackages}

      makeBinaryWrapper "$out/libexec/hpkg" "$out/bin/hpkg" \
        --inherit-argv0 --prefix PATH : ${
          lib.makeBinPath (
            appPackages
            ++ [
              zstd
              gnutar
              coreutils
            ]
          )
        }
    '';

  passthru.targetPkgs = [
    go
    gcc
    xorg.xorgproto
    util-linux

    # for go generate
    wayland-protocols
    wayland-scanner
  ]
  ++ buildInputs
  ++ nativeBuildInputs;
}
