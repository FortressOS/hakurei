{
  writeShellScript,
  writeText,
  callPackage,

  version,
}:
name: want:
writeShellScript "fortify-${name}-check-sandbox-script" ''
  set -e
  ${callPackage ./assert.nix { inherit version; }}/bin/test \
    ${writeText "fortify-${name}-want.json" (builtins.toJSON want)}
  touch /tmp/sandbox-ok
''
