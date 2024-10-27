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
  version = "0.0.9";

  src = ./.;
  vendorHash = null;

  ldflags = [
    "-s"
    "-w"
    "-X"
    "main.Version=v${version}"
    "-X"
    "main.FortifyPath=${placeholder "out"}/bin/fortify"
  ];

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
  '';
}
