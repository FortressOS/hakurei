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

      buildPackage = forAllSystems (
        system:
        nixpkgsFor.${system}.callPackage (
          import ./bundle.nix {
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
          inherit (self.packages.${system}) fortify;
          pkgs = nixpkgsFor.${system};
        in
        {
          default = self.packages.${system}.fortify;
          fortify = pkgs.callPackage ./package.nix { };

          dist =
            pkgs.runCommand "${fortify.name}-dist" { inherit (self.devShells.${system}.default) buildInputs; }
              ''
                # go requires XDG_CACHE_HOME for the build cache
                export XDG_CACHE_HOME="$(mktemp -d)"

                # get a different workdir as go does not like /build
                cd $(mktemp -d) && cp -r ${fortify.src}/. . && chmod -R +w .

                export FORTIFY_VERSION="v${fortify.version}"
                ./dist/release.sh && mkdir $out && cp -v "dist/fortify-$FORTIFY_VERSION.tar.gz"* $out
              '';

          fhs = pkgs.buildFHSEnv {
            pname = "fortify-fhs";
            inherit (fortify) version;
            targetPkgs =
              pkgs:
              with pkgs;
              [
                go
                gcc
                pkg-config
                wayland-scanner
              ]
              ++ (
                with pkgs.pkgsStatic;
                [
                  musl
                  libffi
                  acl
                  wayland
                  wayland-protocols
                ]
                ++ (with xorg; [
                  libxcb
                  libXau
                  libXdmcp

                  xorgproto
                ])
              );
            extraOutputsToInstall = [ "dev" ];
            profile = ''
              export PKG_CONFIG_PATH="/usr/share/pkgconfig:$PKG_CONFIG_PATH"
            '';
          };
        }
      );

      devShells = forAllSystems (
        system:
        let
          inherit (self.packages.${system}) fortify fhs;
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs =
              with pkgs;
              [
                go
                gcc
              ]
              ++ fortify.buildInputs
              ++ fortify.nativeBuildInputs;
          };

          fhs = fhs.env;

          withPackage = nixpkgsFor.${system}.mkShell {
            buildInputs = [ self.packages.${system}.fortify ] ++ self.devShells.${system}.default.buildInputs;
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
        }
      );
    };
}
