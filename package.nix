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

  # for fpkg
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
  pname = "fortify";
  version = "0.3.1";

  src = builtins.path {
    name = "${pname}-src";
    path = lib.cleanSource ./.;
    filter = path: type: !(type == "regular" && (lib.hasSuffix ".nix" path || lib.hasSuffix ".py" path)) && !(type == "directory" && lib.hasSuffix "/test" path) && !(type == "directory" && lib.hasSuffix "/cmd/fsu" path);
  };
  vendorHash = null;

  ldflags =
    lib.attrsets.foldlAttrs
      (
        ldflags: name: value:
        ldflags ++ [ "-X git.gensokyo.uk/security/fortify/internal.${name}=${value}" ]
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
        fsu = "/run/wrappers/bin/fsu";
      };

  # nix build environment does not allow acls
  env.GO_TEST_SKIP_ACL = 1;

  buildInputs =
    [
      libffi
      libseccomp
      acl
      wayland
      wayland-protocols
    ]
    ++ (with xorg; [
      libxcb
      libXau
      libXdmcp
    ]);

  nativeBuildInputs = [
    pkg-config
    wayland-scanner
    makeBinaryWrapper
  ];

  preBuild = ''
    HOME="$(mktemp -d)" PATH="${pkg-config}/bin:$PATH" go generate ./...
  '';

  postInstall =
    let
      appPackages = [
        glibc
        xdg-dbus-proxy
      ];
    in
    ''
      install -D --target-directory=$out/share/zsh/site-functions comp/*

      mkdir "$out/libexec"
      mv "$out"/bin/* "$out/libexec/"

      makeBinaryWrapper "$out/libexec/fortify" "$out/bin/fortify" \
        --inherit-argv0 --prefix PATH : ${lib.makeBinPath appPackages}

      makeBinaryWrapper "$out/libexec/fpkg" "$out/bin/fpkg" \
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

  passthru.targetPkgs =
    [
      go
      gcc
      xorg.xorgproto
      util-linux
    ]
    ++ buildInputs
    ++ nativeBuildInputs;
}
