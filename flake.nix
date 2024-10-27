{
  description = "fortify sandbox tool and nixos module";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable-small";
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [
        "aarch64-linux"
        "i686-linux"
        "x86_64-linux"
      ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      nixosModules.fortify = import ./nixos.nix;

      packages = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = self.packages.${system}.fortify;

          fortify = pkgs.callPackage ./package.nix { };
        }
      );

      devShells = forAllSystems (system: {
        default = nixpkgsFor.${system}.mkShell {
          buildInputs = with nixpkgsFor.${system}; self.packages.${system}.fortify.buildInputs;
        };

        withPackage = nixpkgsFor.${system}.mkShell {
          buildInputs =
            with nixpkgsFor.${system};
            self.packages.${system}.fortify.buildInputs ++ [ self.packages.${system}.fortify ];
        };
      });
    };
}
