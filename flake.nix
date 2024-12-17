{
  description = "fortify sandbox tool and nixos module";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11-small";

    home-manager = {
      url = "github:nix-community/home-manager/release-24.11";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs =
    {
      self,
      nixpkgs,
      home-manager,
    }:
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

      checks = forAllSystems (
        system:
        let
          pkgs = nixpkgsFor.${system};

          inherit (pkgs)
            runCommandLocal
            callPackage
            nixfmt-rfc-style
            deadnix
            statix
            ;
        in
        {
          check-formatting =
            runCommandLocal "check-formatting" { nativeBuildInputs = [ nixfmt-rfc-style ]; }
              ''
                cd ${./.}

                echo "running nixfmt..."
                nixfmt --check .

                touch $out
              '';

          check-lint =
            runCommandLocal "check-lint"
              {
                nativeBuildInputs = [
                  deadnix
                  statix
                ];
              }
              ''
                cd ${./.}

                echo "running deadnix..."
                deadnix --fail

                echo "running statix..."
                statix check .

                touch $out
              '';

          nixos-tests = callPackage ./test.nix { inherit system self home-manager; };
        }
      );

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

        fhs = nixpkgsFor.${system}.buildFHSEnv {
          pname = "fortify-fhs";
          inherit (self.packages.${system}.fortify) version;
          targetPkgs =
            pkgs: with pkgs; [
              go
              gcc
              pkg-config
              acl
              wayland
              wayland-scanner
              wayland-protocols
              xorg.libxcb
            ];
          extraOutputsToInstall = [ "dev" ];
          profile = ''
            export PKG_CONFIG_PATH="/usr/share/pkgconfig:$PKG_CONFIG_PATH"
          '';
        };

        withPackage = nixpkgsFor.${system}.mkShell {
          buildInputs =
            with nixpkgsFor.${system};
            self.packages.${system}.fortify.buildInputs ++ [ self.packages.${system}.fortify ];
        };

        generateDoc =
          let
            pkgs = nixpkgsFor.${system};
            inherit (pkgs) lib;

            doc =
              let
                eval = lib.evalModules {
                  specialArgs = {
                    inherit pkgs;
                  };
                  modules = [ ./options.nix ];
                };
                cleanEval = lib.filterAttrsRecursive (n: _: n != "_module") eval;
              in
              pkgs.nixosOptionsDoc { inherit (cleanEval) options; };
            docText = pkgs.runCommand "fortify-module-docs.md" { } ''
              cat ${doc.optionsCommonMark} > $out
              sed -i '/*Declared by:*/,+1 d' $out
            '';
          in
          nixpkgsFor.${system}.mkShell {
            shellHook = ''
              exec cat ${docText} > options.md
            '';
          };
      });
    };
}
