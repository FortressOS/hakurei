{
  lib,
  buildPackage,
  foot,
  wayland-utils,
  inconsolata,
}:

buildPackage {
  name = "foot";
  inherit (foot) version;

  identity = 2;
  id = "org.codeberg.dnkl.foot";

  modules = [
    {
      home.packages = [
        foot

        # For wayland-info:
        wayland-utils
      ];
    }
  ];

  nixosModules = [
    {
      # To help with OCR:
      environment.etc."xdg/foot/foot.ini".text = lib.generators.toINI { } {
        main = {
          font = "inconsolata:size=14";
        };
        colors = rec {
          foreground = "000000";
          background = "ffffff";
          regular2 = foreground;
        };
      };

      fonts.packages = [ inconsolata ];
    }
  ];

  script = ''
    exec foot "$@"
  '';
}
