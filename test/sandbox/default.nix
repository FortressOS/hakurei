{
  lib,
  nixosTest,

  self,
  withRace ? false,
}:

nixosTest {
  name = "hakurei-sandbox" + (if withRace then "-race" else "");
  nodes.machine =
    { options, pkgs, ... }:
    {
      # Run with Go race detector:
      environment.hakurei = lib.mkIf withRace rec {
        # race detector does not support static linking
        package = (pkgs.callPackage ../../package.nix { }).overrideAttrs (previousAttrs: {
          GOFLAGS = previousAttrs.GOFLAGS ++ [ "-race" ];
        });
        hsuPackage = options.environment.hakurei.hsuPackage.default.override { hakurei = package; };
      };

      imports = [
        ./configuration.nix

        self.nixosModules.hakurei
        self.inputs.home-manager.nixosModules.home-manager
      ];
    };

  # adapted from nixos sway integration tests

  # testScriptWithTypes:49: error: Cannot call function of unknown type
  #           (machine.succeed if succeed else machine.execute)(
  #           ^
  # Found 1 error in 1 file (checked 1 source file)
  skipTypeCheck = true;
  testScript = builtins.readFile ./test.py;
}
