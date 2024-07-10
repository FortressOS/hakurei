{
  description = "ego development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/24.05";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs supportedSystems (system: f system);
    in
    {
      devShells = forAllSystems
        (system:
          let
            pkgs = import nixpkgs {
              inherit system;
            };
          in
          {
            default = with pkgs; mkShell
              {
                packages = [
                  clang
                  acl
                  (pkgs.writeShellScriptBin "build" ''
                    go build -v -ldflags '-s -w -X main.Version=flake'
                  '')
                ];
              };
          }
        );
    };
}