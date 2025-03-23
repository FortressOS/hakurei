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

  ent = fsname: dir: type: opts: freq: passno: {
    inherit
      fsname
      dir
      type
      opts
      freq
      passno
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
          ;
      };
    in
    {
      name = "check-sandbox-${tc.name}";
      verbose = true;
      inherit (tc) tty;
      share = foot;
      packages = [ ];
      command = "${checkSandbox tc.name tc.want} > /dev/console";
      extraPaths = [
        {
          src = "/proc/mounts";
          dst = "/.fortify/mounts";
        }
      ];
    };
in
{
  preset = callTestCase ./preset.nix;
  tty = callTestCase ./tty.nix;
}
