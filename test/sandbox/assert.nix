{
  writeText,
  buildGoModule,

  version,
  name,
  want,
}:
let
  wantFile = writeText "fortify-${name}-want.json" (builtins.toJSON want);
  mainFile = writeText "main.go" ''
    package main

    import "os"
    import "git.gensokyo.uk/security/fortify/test/sandbox"

    func main() { (&sandbox.T{FS: os.DirFS("/"), PMountsPath: "/.fortify/mounts"}).MustCheckFile("${wantFile}") }
  '';
in
buildGoModule {
  pname = "fortify-${name}-check-sandbox";
  inherit version;

  src = ../.;
  vendorHash = null;

  preBuild = ''
    go mod init git.gensokyo.uk/security/fortify/test >& /dev/null
    cp ${mainFile} main.go
  '';
}
