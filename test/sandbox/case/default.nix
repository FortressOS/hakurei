{
  lib,
  callPackage,
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

  checkSandbox = callPackage ../. { inherit version; };

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
      command = builtins.toString (checkSandbox tc.name tc.want);
    };
in
{
  preset = callTestCase ./preset.nix;
  tty = callTestCase ./tty.nix;
  mapuid = callTestCase ./mapuid.nix;
}
