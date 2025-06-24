{
  lib,
  buildGoModule,
  pkg-config,
  util-linux,

  version,
}:
buildGoModule rec {
  pname = "check-sandbox";
  inherit version;

  src = builtins.path {
    name = "${pname}-src";
    path = lib.cleanSource ../.;
    filter = path: type: (type == "directory") || (type == "regular" && lib.hasSuffix ".go" path);
  };
  vendorHash = null;

  buildInputs = [ util-linux ];
  nativeBuildInputs = [ pkg-config ];

  preBuild = ''
    go mod init git.gensokyo.uk/security/hakurei/test/sandbox >& /dev/null
  '';

  postInstall = ''
    mv $out/bin/tool $out/bin/hakurei-test
  '';
}
