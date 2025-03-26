{
  lib,
  callPackage,
  writeText,
  foot,

  version,
}:
let
  fs = mode: dir: data: {
    mode = lib.fromHexString mode;
    inherit
      dir
      data
      ;
  };

  ignore = "//ignore";

  ent = root: target: vfs_optstr: fstype: source: fs_optstr: {
    id = -1;
    parent = -1;
    inherit
      root
      target
      vfs_optstr
      fstype
      source
      fs_optstr
      ;
  };

  checkSandbox = callPackage ../assert.nix { inherit version; };

  callTestCase =
    path:
    let
      tc = import path {
        inherit
          fs
          ent
          ignore
          ;
      };
    in
    {
      name = "check-sandbox-${tc.name}";
      verbose = true;
      inherit (tc) tty mapRealUid;
      share = foot;
      packages = [ ];
      path = "${checkSandbox}/bin/test";
      args = [
        "test"
        (toString (writeText "fortify-${tc.name}-want.json" (builtins.toJSON tc.want)))
      ];
    };
in
{
  preset = callTestCase ./preset.nix;
  tty = callTestCase ./tty.nix;
  mapuid = callTestCase ./mapuid.nix;
}
