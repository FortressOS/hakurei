{
  writeShellScript,
  callPackage,

  version,
}:
writeShellScript "check-sandbox" ''
  set -e
  ${callPackage ./mount.nix { inherit version; }}/bin/test
  ${callPackage ./fs.nix { inherit version; }}/bin/test

  touch /tmp/sandbox-ok
''
