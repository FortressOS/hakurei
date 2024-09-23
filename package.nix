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
  version = "0.0.0-beta.4";

  src = ./.;
  vendorHash = null;

  ldflags = [
    "-s"
    "-w"
    "-X"
    "main.Version=v${version}"
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
  '';
}
