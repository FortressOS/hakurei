{
  lib,
  nixosTest,
  buildFHSEnv,
  writeShellScriptBin,

  system,
  self,
  withRace ? false,
}:

nixosTest {
  name = "fortify" + (if withRace then "-race" else "");
  nodes.machine =
    { options, pkgs, ... }:
    let
      fhs =
        let
          fortify = options.environment.fortify.package.default;
        in
        buildFHSEnv {
          pname = "fortify-fhs";
          inherit (fortify) version;
          targetPkgs = _: fortify.targetPkgs;
          extraOutputsToInstall = [ "dev" ];
          profile = ''
            export PKG_CONFIG_PATH="/usr/share/pkgconfig:$PKG_CONFIG_PATH"
          '';
        };
    in
    {
      environment.systemPackages = [
        # For go tests:
        (writeShellScriptBin "fortify-test" ''
          cd ${self.packages.${system}.fortify.src}
          ${fhs}/bin/fortify-fhs -c \
            'go test ${if withRace then "-race" else "-count 16"} ./...' \
            &> /tmp/fortify-test.log && \
            touch /tmp/fortify-test-ok
          touch /tmp/fortify-test-done
        '')
      ];

      # Run with Go race detector:
      environment.fortify = lib.mkIf withRace rec {
        # race detector does not support static linking
        package = (pkgs.callPackage ../package.nix { }).overrideAttrs (previousAttrs: {
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
