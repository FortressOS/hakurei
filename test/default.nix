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
  name = "hakurei" + (if withRace then "-race" else "");
  nodes.machine =
    { options, pkgs, ... }:
    let
      fhs =
        let
          hakurei = options.environment.hakurei.package.default;
        in
        buildFHSEnv {
          pname = "hakurei-fhs";
          inherit (hakurei) version;
          targetPkgs = _: hakurei.targetPkgs;
          extraOutputsToInstall = [ "dev" ];
          profile = ''
            export PKG_CONFIG_PATH="/usr/share/pkgconfig:$PKG_CONFIG_PATH"
          '';
        };
    in
    {
      environment.systemPackages = [
        # For go tests:
        (writeShellScriptBin "hakurei-test" ''
          cd ${self.packages.${system}.hakurei.src}
          ${fhs}/bin/hakurei-fhs -c \
            'go test ${if withRace then "-race" else "-count 16"} ./...' \
            &> /tmp/hakurei-test.log && \
            touch /tmp/hakurei-test-ok
          touch /tmp/hakurei-test-done
        '')
      ];

      # Run with Go race detector:
      environment.hakurei = lib.mkIf withRace rec {
        # race detector does not support static linking
        package = (pkgs.callPackage ../package.nix { }).overrideAttrs (previousAttrs: {
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
