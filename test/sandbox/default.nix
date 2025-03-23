{
  writeShellScript,
  callPackage,

  name,
  version,
  want,
}:
writeShellScript "fortify-${name}-check-sandbox-script" ''
  set -e
  ${callPackage ./assert.nix { inherit name version want; }}/bin/test
  touch /tmp/sandbox-ok
''
