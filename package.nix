{
  lib,
  buildGoModule,
  makeBinaryWrapper,
  xdg-dbus-proxy,
  bubblewrap,
  acl,
  xorg,
}:

buildGoModule rec {
  pname = "fortify";
  version = "0.2.0";

  src = ./.;
  vendorHash = null;

  ldflags =
    lib.attrsets.foldlAttrs
      (
        ldflags: name: value:
        ldflags
        ++ [
          "-X"
          "git.ophivana.moe/security/fortify/internal.${name}=${value}"
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

  buildInputs = [
    acl
    xorg.libxcb
  ];

  nativeBuildInputs = [ makeBinaryWrapper ];

  postInstall = ''
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
