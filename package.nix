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
  version = "0.0.11";

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
      ]
      {
        Version = "v${version}";
        Fsu = "/run/wrappers/bin/fsu";
        Fshim = "${placeholder "out"}/bin/.fshim";
        Finit = "${placeholder "out"}/bin/.finit";
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

    mv $out/bin/fsu $out/bin/.fsu
    mv $out/bin/fshim $out/bin/.fshim
    mv $out/bin/finit $out/bin/.finit
  '';
}
