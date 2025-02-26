{
  lib,
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

  ldflags =
    lib.attrsets.foldlAttrs
      (
        ldflags: name: value:
        ldflags ++ [ "-X main.${name}=${value}" ]
      )
      [ "-s -w" ]
      {
        fmain = "${fortify}/libexec/fortify";
        fpkg = "${fortify}/libexec/fpkg";
      };
}
