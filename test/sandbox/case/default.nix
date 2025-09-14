system: lib: testProgram:
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

  importTestCase =
    path:
    import path {
      inherit
        fs
        ent
        ignore
        system
        ;
    };

  callTestCase =
    path: identity:
    let
      tc = importTestCase path;
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
        userns
        hostAbstract
        ;
      enablements = {
        inherit (tc) x11;
      };
      share = testProgram;
      packages = [ ];
      path = "${testProgram}/bin/hakurei-test";
      args = [
        "test"
        "-t"
        (toString (builtins.toFile "hakurei-${tc.name}-want.json" (builtins.toJSON tc.want)))
        "-s"
        tc.expectedFilter.${system}
      ];

      extraPaths =
        if tc.useCommonPaths then
          [ ]
        else
          [
            {
              type = "bind";
              src = "/var/tmp";
              write = true;
            }
          ];
    };

  testCaseName = name: "cat.gensokyo.hakurei.test." + name;
in
{
  apps = {
    ${testCaseName "preset"} = callTestCase ./preset.nix 1;
    ${testCaseName "tty"} = callTestCase ./tty.nix 2;
    ${testCaseName "mapuid"} = callTestCase ./mapuid.nix 3;
    ${testCaseName "device"} = callTestCase ./device.nix 4;
    ${testCaseName "pdlike"} = callTestCase ./pdlike.nix 5;
  };

  pd = importTestCase ./pd.nix;
}
