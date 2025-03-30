{
  lib,
  nixosTest,

  self,
  withRace ? false,
}:

nixosTest {
  name = "fortify-sandbox" + (if withRace then "-race" else "");
  nodes.machine =
    { options, pkgs, ... }:
    {
      # Run with Go race detector:
      environment.fortify = lib.mkIf withRace rec {
        # race detector does not support static linking
        package = (pkgs.callPackage ../../package.nix { }).overrideAttrs (previousAttrs: {
          GOFLAGS = previousAttrs.GOFLAGS ++ [ "-race" ];
        });
        fsuPackage = options.environment.fortify.fsuPackage.default.override { fortify = package; };
      };

      imports = [
        ./configuration.nix

        self.nixosModules.fortify
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
