{
  lib,
  buildGoModule,
  xdg-dbus-proxy,
  bubblewrap,
  pkgsStatic,
  pkg-config,
  wayland-scanner,
}:

buildGoModule rec {
  pname = "fortify";
  version = "0.2.9";

  src = builtins.path {
    name = "fortify-src";
    path = lib.cleanSource ./.;
    filter = path: type: !(type != "directory" && lib.hasSuffix ".nix" path);
  };
  vendorHash = null;

  ldflags =
    lib.attrsets.foldlAttrs
      (
        ldflags: name: value:
        ldflags ++ [ "-X git.gensokyo.uk/security/fortify/internal.${name}=${value}" ]
      )
      [
        "-s -w"
        "-extldflags '-static'"
        "-X main.Fmain=${placeholder "out"}/libexec/fortify"
        "-X main.Fshim=${placeholder "out"}/libexec/fshim"
      ]
      {
        Version = "v${version}";
        Fsu = "/run/wrappers/bin/fsu";
        Finit = "${placeholder "out"}/libexec/finit";
        Fortify = "${placeholder "out"}/bin/fortify";
      };

  # nix build environment does not allow acls
  GO_TEST_SKIP_ACL = 1;

  buildInputs =
    # cannot find a cleaner way to do this
    with pkgsStatic;
    [
      musl
      libffi
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
    pkgsStatic.makeBinaryWrapper
  ];

  preConfigure = ''
    HOME=$(mktemp -d) go generate ./...
  '';

  postInstall = ''
    install -D --target-directory=$out/share/zsh/site-functions comp/*

    mkdir "$out/libexec"
    mv "$out"/bin/* "$out/libexec/"

    makeBinaryWrapper "$out/libexec/fortify" "$out/bin/fortify" \
      --inherit-argv0 --prefix PATH : ${
        lib.makeBinPath [
          bubblewrap
          xdg-dbus-proxy
        ]
      }
  '';
}
