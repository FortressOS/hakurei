{
  lib,
  buildGoModule,
  makeBinaryWrapper,
  xdg-dbus-proxy,
  bubblewrap,
  pkg-config,
  acl,
  wayland,
  wayland-scanner,
  wayland-protocols,
  xorg,
}:

buildGoModule rec {
  pname = "fortify";
  version = "0.2.8";

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
        ldflags
        ++ [
          "-X"
          "git.gensokyo.uk/security/fortify/internal.${name}=${value}"
        ]
      )
      [
        "-s"
        "-w"
        "-X"
        "main.Fmain=${placeholder "out"}/libexec/fortify"
        "-X"
        "main.Fshim=${placeholder "out"}/libexec/fshim"
      ]
      {
        Version = "v${version}";
        Fsu = "/run/wrappers/bin/fsu";
        Finit = "${placeholder "out"}/libexec/finit";
      };

  # nix build environment does not allow acls
  GO_TEST_SKIP_ACL = 1;

  buildInputs = [
    acl
    wayland
    wayland-protocols
    xorg.libxcb
  ];

  nativeBuildInputs = [
    pkg-config
    wayland-scanner
    makeBinaryWrapper
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
