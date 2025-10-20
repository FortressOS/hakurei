{
  lib,
  buildGoModule,
  hakurei ? abort "hakurei package required",
}:

buildGoModule {
  pname = "${hakurei.pname}-hsu";
  inherit (hakurei) version;

  src = ./.;
  inherit (hakurei) vendorHash;
  env.CGO_ENABLED = 0;

  preBuild = ''
    go mod init hsu >& /dev/null
  '';

  ldflags = lib.attrsets.foldlAttrs (
    ldflags: name: value:
    ldflags ++ [ "-X main.${name}=${value}" ]
  ) [ "-s -w" ] { hakureiPath = "${hakurei}/libexec/hakurei"; };
}
