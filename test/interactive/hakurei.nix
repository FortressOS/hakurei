{ pkgs, ... }:
{
  environment.hakurei = {
    enable = true;
    stateDir = "/var/lib/hakurei";
    users.alice = 0;
    apps = {
      "cat.gensokyo.extern.foot.noEnablements" = {
        name = "ne-foot";
        identity = 1;
        shareUid = true;
        verbose = true;
        share = pkgs.foot;
        packages = [ pkgs.foot ];
        command = "foot";
        enablements = {
          dbus = false;
          pulse = false;
        };
      };
    };

    extraHomeConfig.home.stateVersion = "23.05";
  };
}
