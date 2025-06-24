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
    path: identity:
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
      inherit identity;
      verbose = true;
      inherit (tc)
        tty
        device
        mapRealUid
        useCommonPaths
        ;
      share = testProgram;
      packages = [ ];
      path = "${testProgram}/bin/hakurei-test";
      args = [
        "test"
        (toString (builtins.toFile "hakurei-${tc.name}-want.json" (builtins.toJSON tc.want)))
      ];
    };

  testCaseName = name: "cat.gensokyo.hakurei.test." + name;
in
{
  ${testCaseName "preset"} = callTestCase ./preset.nix 1;
  ${testCaseName "tty"} = callTestCase ./tty.nix 2;
  ${testCaseName "mapuid"} = callTestCase ./mapuid.nix 3;
  ${testCaseName "device"} = callTestCase ./device.nix 4;
}
