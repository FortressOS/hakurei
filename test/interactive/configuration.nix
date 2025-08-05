{ pkgs, ... }:
{
  system.stateVersion = "23.05";

  users.users = {
    alice = {
      isNormalUser = true;
      description = "Alice Foobar";
      password = "foobar";
      uid = 1000;
      extraGroups = [ "wheel" ];
    };
    untrusted = {
      isNormalUser = true;
      description = "Untrusted user";
      password = "foobar";
      uid = 1001;
    };
  };

  home-manager.users.alice.home.stateVersion = "24.11";

  security = {
    sudo.wheelNeedsPassword = false;
    rtkit.enable = true;
  };

  services = {
    getty.autologinUser = "alice";
    pipewire = {
      enable = true;
      alsa.enable = true;
      alsa.support32Bit = true;
      pulse.enable = true;
      jack.enable = true;
    };
  };

  environment.variables = {
    SWAYSOCK = "/tmp/sway-ipc.sock";
    WLR_RENDERER = "pixman";
  };

  programs = {
    sway.enable = true;

    bash.loginShellInit = ''
      if [ "$(tty)" = "/dev/tty1" ]; then
        set -e

        mkdir -p ~/.config/sway
        (sed s/Mod4/Mod1/ /etc/sway/config &&
        echo 'output * bg ${pkgs.nixos-artwork.wallpapers.simple-light-gray.gnomeFilePath} fill') > ~/.config/sway/config

        sway --validate
        systemd-cat --identifier=session sway && touch /tmp/sway-exit-ok
      fi
    '';
  };
}
