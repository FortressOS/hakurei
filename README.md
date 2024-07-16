ego (the Go side)
=================

[![Go Reference](https://pkg.go.dev/badge/git.ophivana.moe/cat/ego.svg)](https://pkg.go.dev/git.ophivana.moe/cat/ego)

> Do all your games need access to your documents, browser history, SSH private keys?
>
> ... No? Just run `ego steam`!

**Ego** is a tool to run Linux desktop applications under a different local user. Currently
integrates with Wayland, Xorg, PulseAudio and xdg-desktop-portal. You may think of it as `xhost`
for Wayland and PulseAudio. This is done using filesystem ACLs and X11 host access control.

Disclaimer: **DO NOT RUN UNTRUSTED PROGRAMS VIA EGO.** However, using ego is more secure than
running applications directly under your primary user.

Differences
-----------
* Written in Go
* Tracks process states
* Cleans up after last process exits
* Argv preservation in machinectl mode
* Has no dependencies other than the two C libraries

Manual setup
------------
Ego aims to come with sane defaults and be easy to set up.

**Requirements:**
* Sudo
* A C compiler
* [Go](https://go.dev/doc/install)
* `libacl.so` library (Debian/Ubuntu: libacl1-dev; Fedora: libacl-devel; Arch: acl)
* `libxcb.so` library (Debian/Ubuntu: libxcb1-dev; Fedora: libxcb-devel; Arch: libxcb)

**Recommended:** (Not needed when using `--sudo` mode, but some desktop functionality may not work).
* `machinectl` command (Debian/Ubuntu/Fedora: systemd-container; Arch: systemd)
* `xdg-desktop-portal-gtk` (Debian/Ubuntu/Fedora/Arch: xdg-desktop-portal-gtk)

**Installation:**

1. Run in repository worktree:

       go build -v -ldflags '-s -w'
       sudo cp ego /usr/local/bin/

2. Create local user named "ego": <sup>[1]</sup>

       sudo useradd ego --uid 155 --create-home

3. That's all, try it:

       ego xdg-open .

[1] No extra groups are needed by the ego user.
UID below 1000 hides this user on the login screen.

### Avoid password prompt
If using "machinectl" mode (default if available), you need the rather new systemd version >=247
and polkit >=0.106 to do this securely.

Create file `/etc/polkit-1/rules.d/50-ego-machinectl.rules`, polkit will automatically load it
(replace `$USER` with your own username):

```js
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.machine1.host-shell" &&
        action.lookup("user") == "ego" &&
        subject.user == "$USER") {
            return polkit.Result.YES;
    }
});
```

##### sudo mode
For sudo, add the following to `/etc/sudoers` (replace `$USER` with your own username):

    $USER ALL=(ego) NOPASSWD:ALL

Appendix
--------
Ego is licensed under the MIT License (see the `LICENSE` file).
The original Ego was created by Marti Raudsepp under the repository https://github.com/intgr/ego