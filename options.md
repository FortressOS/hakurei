## environment\.fortify\.enable



Whether to enable fortify\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.package



The fortify package to use\.



*Type:*
package



*Default:*
` <derivation fortify-static-x86_64-unknown-linux-musl-0.4.1> `



## environment\.fortify\.apps

Declaratively configured fortify apps\.



*Type:*
attribute set of (submodule)



*Default:*
` { } `



## environment\.fortify\.apps\.\<name>\.packages



List of extra packages to install via home-manager\.



*Type:*
list of package



*Default:*
` [ ] `



## environment\.fortify\.apps\.\<name>\.args



Custom args\.
Setting this to null will default to script name\.



*Type:*
null or (list of string)



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.capability\.dbus



Whether to proxy D-Bus\.



*Type:*
boolean



*Default:*
` true `



## environment\.fortify\.apps\.\<name>\.capability\.pulse



Whether to share the PulseAudio socket and cookie\.



*Type:*
boolean



*Default:*
` true `



## environment\.fortify\.apps\.\<name>\.capability\.wayland



Whether to share the Wayland socket\.



*Type:*
boolean



*Default:*
` true `



## environment\.fortify\.apps\.\<name>\.capability\.x11



Whether to share the X11 socket and allow connection\.



*Type:*
boolean



*Default:*
` false `



## environment\.fortify\.apps\.\<name>\.command



Command to run as the target user\.
Setting this to null will default command to launcher name\.
Has no effect when script is set\.



*Type:*
null or string



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.dbus\.session



D-Bus session bus custom configuration\.
Setting this to null will enable built-in defaults\.



*Type:*
null or (function that evaluates to a(n) anything)



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.dbus\.system



D-Bus system bus custom configuration\.
Setting this to null will disable the system bus proxy\.



*Type:*
null or anything



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.devel



Whether to enable debugging-related kernel interfaces\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.device



Whether to enable access to all devices\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.env



Environment variables to set for the initial process in the sandbox\.



*Type:*
null or (attribute set of string)



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.extraConfig



Extra home-manager configuration\.



*Type:*
anything



*Default:*
` { } `



## environment\.fortify\.apps\.\<name>\.extraPaths



Extra paths to make available to the container\.



*Type:*
list of (submodule)



*Default:*
` [ ] `



## environment\.fortify\.apps\.\<name>\.extraPaths\.\*\.dev



Whether to enable use of device files\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.extraPaths\.\*\.dst



Mount point in container, same as src if null\.



*Type:*
null or string



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.extraPaths\.\*\.require



Whether to enable start failure if the bind mount cannot be established for any reason\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.extraPaths\.\*\.src



Host filesystem path to make available to the container\.



*Type:*
string



## environment\.fortify\.apps\.\<name>\.extraPaths\.\*\.write



Whether to enable mounting path as writable\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.gpu



Target process GPU and driver access\.
Setting this to null will enable GPU whenever X or Wayland is enabled\.



*Type:*
null or boolean



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.groups



List of groups to inherit from the privileged user\.



*Type:*
list of string



*Default:*
` [ ] `



## environment\.fortify\.apps\.\<name>\.identity



Application identity\. Identity 0 is reserved for system services\.



*Type:*
integer between 1 and 9999 (both inclusive)



## environment\.fortify\.apps\.\<name>\.insecureWayland



Whether to enable direct access to the Wayland socket\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.mapRealUid



Whether to enable mapping to priv-user uid\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.multiarch



Whether to enable multiarch kernel-level support\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.name



Name of the app’s launcher script\.



*Type:*
string



## environment\.fortify\.apps\.\<name>\.net



Whether to enable network access\.



*Type:*
boolean



*Default:*
` true `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.nix



Whether to enable nix daemon access\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.path



Custom executable path\.
Setting this to null will default to the start script\.



*Type:*
null or string



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.script



Application launch script\.



*Type:*
null or string



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.share



Package containing share files\.
Setting this to null will default package name to wrapper name\.



*Type:*
null or package



*Default:*
` null `



## environment\.fortify\.apps\.\<name>\.shareUid



Whether to enable sharing identity with another application\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.tty



Whether to enable access to the controlling terminal\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.useCommonPaths



Whether to enable common extra paths\.



*Type:*
boolean



*Default:*
` true `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.userns



Whether to enable user namespace creation\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.apps\.\<name>\.verbose



Whether to enable launchers with verbose output\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.commonPaths



Common extra paths to make available to the container\.



*Type:*
list of (submodule)



*Default:*
` [ ] `



## environment\.fortify\.commonPaths\.\*\.dev



Whether to enable use of device files\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.commonPaths\.\*\.dst



Mount point in container, same as src if null\.



*Type:*
null or string



*Default:*
` null `



## environment\.fortify\.commonPaths\.\*\.require



Whether to enable start failure if the bind mount cannot be established for any reason\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.commonPaths\.\*\.src



Host filesystem path to make available to the container\.



*Type:*
string



## environment\.fortify\.commonPaths\.\*\.write



Whether to enable mounting path as writable\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.fortify\.extraHomeConfig



Extra home-manager configuration to merge with all target users\.



*Type:*
anything



## environment\.fortify\.fsuPackage



The fsu package to use\.



*Type:*
package



*Default:*
` <derivation fortify-fsu-0.4.1> `



## environment\.fortify\.stateDir



The state directory where app home directories are stored\.



*Type:*
string



## environment\.fortify\.users



Users allowed to spawn fortify apps and their corresponding fortify fid\.



*Type:*
attribute set of integer between 0 and 99 (both inclusive)


