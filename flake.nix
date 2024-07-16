{
  description = "ego development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/24.05";
  };

  outputs =
    { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs supportedSystems (system: f system);
    in
    {
      devShells = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          default =
            let
              inherit (pkgs)
                mkShell
                buildGoModule
                acl
                xorg
                ;
            in
            mkShell {
              packages = [
                (buildGoModule rec {
                  pname = "ego";
                  version = "flake";

                  src = ./.;
                  vendorHash = null; # we have no dependencies :3

                  ldflags = [
                    "-s"
                    "-w"
                    "-X"
                    "main.Version=v${version}"
                  ];

                  buildInputs = [
                    acl
                    xorg.libxcb
                  ];
                })
              ];
            };
        }
      );
    };
}
