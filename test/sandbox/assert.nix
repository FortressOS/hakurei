{
  writeText,
  buildGoModule,

  version,
}:
buildGoModule {
  pname = "check-sandbox";
  inherit version;

  src = ../.;
  vendorHash = null;

  preBuild = ''
    go mod init git.gensokyo.uk/security/fortify/test >& /dev/null
    cp ${writeText "main.go" ''
      package main

      import "os"
      import "git.gensokyo.uk/security/fortify/test/sandbox"

      func main() { (&sandbox.T{FS: os.DirFS("/"), PMountsPath: "/.fortify/mounts"}).MustCheckFile(os.Args[1]) }
    ''} main.go
  '';
}
