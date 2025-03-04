{
  writeText,
  buildGoModule,

  version,
}:
let
  mainFile = writeText "main.go" ''
    package main

    import "git.gensokyo.uk/security/fortify/test/sandbox"

    func main() { sandbox.MustAssertSeccomp() }
  '';
in
buildGoModule {
  pname = "check-seccomp";
  inherit version;

  src = ../.;
  vendorHash = null;

  preBuild = ''
    go mod init git.gensokyo.uk/security/fortify/test >& /dev/null
    cp ${mainFile} main.go
  '';
}
