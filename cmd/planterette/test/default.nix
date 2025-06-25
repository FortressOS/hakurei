{
  nixosTest,
  callPackage,

  system,
  self,
}:
let
  buildPackage = self.buildPackage.${system};
in
nixosTest {
  name = "planterette";
  nodes.machine = {
    environment.etc = {
      "foot.pkg".source = callPackage ./foot.nix { inherit buildPackage; };
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
