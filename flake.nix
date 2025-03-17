{
  description = "fortify sandbox tool and nixos module";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";

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
      nixosModules.fortify = import ./nixos.nix self.packages;

      buildPackage = forAllSystems (
        system:
        nixpkgsFor.${system}.callPackage (
          import ./cmd/fpkg/build.nix {
            inherit
              nixpkgsFor
              system
              nixpkgs
              home-manager
              ;
          }
        )
      );

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
          fortify = callPackage ./test { inherit system self; };
          fpkg = callPackage ./cmd/fpkg/test { inherit system self; };
          race = callPackage ./test {
            inherit system self;
            withRace = true;
          };

          formatting = runCommandLocal "check-formatting" { nativeBuildInputs = [ nixfmt-rfc-style ]; } ''
            cd ${./.}

            echo "running nixfmt..."
            nixfmt --width=256 --check .

            touch $out
          '';

          lint =
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
        }
      );

      packages = forAllSystems (
        system:
        let
          inherit (self.packages.${system}) fortify fsu;
          pkgs = nixpkgsFor.${system};
        in
        {
          default = fortify;
          fortify = pkgs.pkgsStatic.callPackage ./package.nix {
            inherit (pkgs)
              # passthru.buildInputs
              go
              gcc

              # nativeBuildInputs
              pkg-config
              wayland-scanner
              makeBinaryWrapper

              # appPackages
              glibc
              bubblewrap
              xdg-dbus-proxy

              # fpkg
              zstd
              gnutar
              coreutils
              ;
          };
          fsu = pkgs.callPackage ./cmd/fsu/package.nix { inherit (self.packages.${system}) fortify; };

          dist = pkgs.runCommand "${fortify.name}-dist" { buildInputs = fortify.targetPkgs ++ [ pkgs.pkgsStatic.musl ]; } ''
            # go requires XDG_CACHE_HOME for the build cache
            export XDG_CACHE_HOME="$(mktemp -d)"

            # get a different workdir as go does not like /build
            cd $(mktemp -d) \
                && cp -r ${fortify.src}/. . \
                && chmod +w cmd && cp -r ${fsu.src}/. cmd/fsu/ \
                && chmod -R +w .

            export FORTIFY_VERSION="v${fortify.version}"
            ./dist/release.sh && mkdir $out && cp -v "dist/fortify-$FORTIFY_VERSION.tar.gz"* $out
          '';
        }
      );

      devShells = forAllSystems (
        system:
        let
          inherit (self.packages.${system}) fortify;
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell { buildInputs = fortify.targetPkgs; };
          withPackage = pkgs.mkShell { buildInputs = [ fortify ] ++ fortify.targetPkgs; };

          generateDoc =
            let
              inherit (pkgs) lib;

              doc =
                let
                  eval = lib.evalModules {
                    specialArgs = {
                      inherit pkgs;
                    };
                    modules = [ (import ./options.nix self.packages) ];
                  };
                  cleanEval = lib.filterAttrsRecursive (n: _: n != "_module") eval;
                in
                pkgs.nixosOptionsDoc { inherit (cleanEval) options; };
              docText = pkgs.runCommand "fortify-module-docs.md" { } ''
                cat ${doc.optionsCommonMark} > $out
                sed -i '/*Declared by:*/,+1 d' $out
              '';
            in
            pkgs.mkShell {
              shellHook = ''
                exec cat ${docText} > options.md
              '';
            };
        }
      );
    };
}
