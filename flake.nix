{
  description = "hakurei container tool and nixos module";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.05";

    home-manager = {
      url = "github:nix-community/home-manager/release-25.05";
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
      nixosModules.hakurei = import ./nixos.nix self.packages;

      buildPackage = forAllSystems (
        system:
        nixpkgsFor.${system}.callPackage (
          import ./cmd/planterette/build.nix {
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
          hakurei = callPackage ./test { inherit system self; };
          race = callPackage ./test {
            inherit system self;
            withRace = true;
          };

          sandbox = callPackage ./test/sandbox { inherit self; };
          sandbox-race = callPackage ./test/sandbox {
            inherit self;
            withRace = true;
          };

          planterette = callPackage ./cmd/planterette/test { inherit system self; };

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
          inherit (self.packages.${system}) hakurei hsu;
          pkgs = nixpkgsFor.${system};
        in
        {
          default = hakurei;
          hakurei = pkgs.pkgsStatic.callPackage ./package.nix {
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
              xdg-dbus-proxy

              # planterette
              zstd
              gnutar
              coreutils
              ;
          };
          hsu = pkgs.callPackage ./cmd/hsu/package.nix { inherit (self.packages.${system}) hakurei; };

          dist = pkgs.runCommand "${hakurei.name}-dist" { buildInputs = hakurei.targetPkgs ++ [ pkgs.pkgsStatic.musl ]; } ''
            # go requires XDG_CACHE_HOME for the build cache
            export XDG_CACHE_HOME="$(mktemp -d)"

            # get a different workdir as go does not like /build
            cd $(mktemp -d) \
                && cp -r ${hakurei.src}/. . \
                && chmod +w cmd && cp -r ${hsu.src}/. cmd/hsu/ \
                && chmod -R +w .

            export HAKUREI_VERSION="v${hakurei.version}"
            ./dist/release.sh && mkdir $out && cp -v "dist/hakurei-$HAKUREI_VERSION.tar.gz"* $out
          '';
        }
      );

      devShells = forAllSystems (
        system:
        let
          inherit (self.packages.${system}) hakurei;
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell { buildInputs = hakurei.targetPkgs; };
          withPackage = pkgs.mkShell { buildInputs = [ hakurei ] ++ hakurei.targetPkgs; };

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
              docText = pkgs.runCommand "hakurei-module-docs.md" { } ''
                cat ${doc.optionsCommonMark} > $out
                sed -i '/*Declared by:*/,+1 d' $out
              '';
            in
            pkgs.mkShell {
              shellHook = ''
                exec cat ${docText} > options.md
              '';
            };

          generateSyscallTable = pkgs.mkShell {
            # this should be made cross-platform via nix
            shellHook = "exec ${pkgs.writeShellScript "generate-syscall-table" ''
              set -e
              ${pkgs.perl}/bin/perl \
                sandbox/seccomp/mksysnum_linux.pl \
                ${pkgs.linuxHeaders}/include/asm/unistd_64.h | \
                ${pkgs.go}/bin/gofmt > \
                sandbox/seccomp/syscall_linux_amd64.go
            ''}";
          };
        }
      );
    };
}
