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
  version = "0.1.0";

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
        "main.Fmain=${placeholder "out"}/bin/.fortify-wrapped"
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
    wrapProgram $out/bin/${pname} --prefix PATH : ${
      lib.makeBinPath [
        bubblewrap
        xdg-dbus-proxy
      ]
    }

    mkdir $out/libexec
    (cd $out/bin && mv fsu fshim finit fuserdb ../libexec/)
  '';
}
