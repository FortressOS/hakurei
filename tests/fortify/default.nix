{
  system,
  self,
  nixosTest,
  writeShellScriptBin,
}:

nixosTest {
  name = "fortify";
  nodes.machine = {
    environment.systemPackages = [
      # For go tests:
      self.packages.${system}.fhs
      (writeShellScriptBin "fortify-src" "echo -n ${self.packages.${system}.fortify.src}")
    ];

    # Run with Go race detector:
    environment.fortify.package =
      let
        inherit (self.packages.${system}) fortify;
      in
      fortify.overrideAttrs (previousAttrs: {
        GOFLAGS = previousAttrs.GOFLAGS ++ [ "-race" ];

        # fsu does not like cgo
        disallowedReferences = previousAttrs.disallowedReferences ++ [ fortify ];
        postInstall =
          previousAttrs.postInstall
          + ''
            cp -a "${fortify}/libexec/fsu" "$out/libexec/fsu"
            sed -i 's:${fortify}:${placeholder "out"}:' "$out/libexec/fsu"
          '';
      });

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
