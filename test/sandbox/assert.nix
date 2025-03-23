{
  writeText,
  buildGoModule,
  pkg-config,
  util-linux,

  version,
}:
buildGoModule {
  pname = "check-sandbox";
  inherit version;

  src = ../.;
  vendorHash = null;

  buildInputs = [ util-linux ];
  nativeBuildInputs = [ pkg-config ];

  preBuild = ''
    go mod init git.gensokyo.uk/security/fortify/test >& /dev/null
    cp ${writeText "main.go" ''
      package main

      import "os"
      import "git.gensokyo.uk/security/fortify/test/sandbox"

      func main() { (&sandbox.T{FS: os.DirFS("/")}).MustCheckFile(os.Args[1]) }
    ''} main.go
  '';
}
