pkgs: version:
let
  inherit (pkgs)
    lib
    writeText
    buildGoModule
    pkg-config
    util-linux
    foot
    ;

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

  checkSandbox = buildGoModule {
    pname = "check-sandbox";
    inherit version;

    src = ../../.;
    vendorHash = null;

    buildInputs = [ util-linux ];
    nativeBuildInputs = [ pkg-config ];

    preBuild = ''
      go mod init git.gensokyo.uk/security/fortify/test >& /dev/null
      cp ${./main.go} main.go
    '';
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
