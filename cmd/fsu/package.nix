{
  buildGoModule,
  fortify ? abort "fortify package required",
}:

buildGoModule {
  pname = "${fortify.pname}-fsu";
  inherit (fortify) version;

  src = ./.;
  inherit (fortify) vendorHash;
  CGO_ENABLED = 0;

  preBuild = ''
    go mod init fsu >& /dev/null
  '';

  ldflags = [ "-X main.Fmain=${fortify}/libexec/fortify" ];
}
