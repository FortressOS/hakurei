lib: testProgram:
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
      inherit (tc)
        tty
        device
        mapRealUid
        useCommonPaths
        ;
      share = testProgram;
      packages = [ ];
      path = "${testProgram}/bin/fortify-test";
      args = [
        "test"
        (toString (builtins.toFile "fortify-${tc.name}-want.json" (builtins.toJSON tc.want)))
      ];
    };
in
{
  preset = callTestCase ./preset.nix;
  tty = callTestCase ./tty.nix;
  mapuid = callTestCase ./mapuid.nix;
  device = callTestCase ./device.nix;
}
