## environment\.hakurei\.enable



Whether to enable hakurei\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.package



The hakurei package to use\.



*Type:*
package



*Default:*
` <derivation hakurei-static-x86_64-unknown-linux-musl-0.0.2> `



## environment\.hakurei\.apps

Declaratively configured hakurei apps\.



*Type:*
attribute set of (submodule)



*Default:*
` { } `



## environment\.hakurei\.apps\.\<name>\.packages



List of extra packages to install via home-manager\.



*Type:*
list of package



*Default:*
` [ ] `



## environment\.hakurei\.apps\.\<name>\.args



Custom args\.
Setting this to null will default to script name\.



*Type:*
null or (list of string)



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.capability\.dbus



Whether to proxy D-Bus\.



*Type:*
boolean



*Default:*
` true `



## environment\.hakurei\.apps\.\<name>\.capability\.pulse



Whether to share the PulseAudio socket and cookie\.



*Type:*
boolean



*Default:*
` true `



## environment\.hakurei\.apps\.\<name>\.capability\.wayland



Whether to share the Wayland socket\.



*Type:*
boolean



*Default:*
` true `



## environment\.hakurei\.apps\.\<name>\.capability\.x11



Whether to share the X11 socket and allow connection\.



*Type:*
boolean



*Default:*
` false `



## environment\.hakurei\.apps\.\<name>\.command



Command to run as the target user\.
Setting this to null will default command to launcher name\.
Has no effect when script is set\.



*Type:*
null or string



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.dbus\.session



D-Bus session bus custom configuration\.
Setting this to null will enable built-in defaults\.



*Type:*
null or (function that evaluates to a(n) anything)



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.dbus\.system



D-Bus system bus custom configuration\.
Setting this to null will disable the system bus proxy\.



*Type:*
null or anything



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.devel



Whether to enable debugging-related kernel interfaces\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.device



Whether to enable access to all devices\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.env



Environment variables to set for the initial process in the sandbox\.



*Type:*
null or (attribute set of string)



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.extraConfig



Extra home-manager configuration\.



*Type:*
anything



*Default:*
` { } `



## environment\.hakurei\.apps\.\<name>\.extraPaths



Extra paths to make available to the container\.



*Type:*
list of (submodule)



*Default:*
` [ ] `



## environment\.hakurei\.apps\.\<name>\.extraPaths\.\*\.dev



Whether to enable use of device files\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.extraPaths\.\*\.dst



Mount point in container, same as src if null\.



*Type:*
null or string



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.extraPaths\.\*\.require



Whether to enable start failure if the bind mount cannot be established for any reason\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.extraPaths\.\*\.src



Host filesystem path to make available to the container\.



*Type:*
string



## environment\.hakurei\.apps\.\<name>\.extraPaths\.\*\.write



Whether to enable mounting path as writable\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.gpu



Target process GPU and driver access\.
Setting this to null will enable GPU whenever X or Wayland is enabled\.



*Type:*
null or boolean



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.groups



List of groups to inherit from the privileged user\.



*Type:*
list of string



*Default:*
` [ ] `



## environment\.hakurei\.apps\.\<name>\.identity



Application identity\. Identity 0 is reserved for system services\.



*Type:*
integer between 1 and 9999 (both inclusive)



## environment\.hakurei\.apps\.\<name>\.insecureWayland



Whether to enable direct access to the Wayland socket\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.mapRealUid



Whether to enable mapping to priv-user uid\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.multiarch



Whether to enable multiarch kernel-level support\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.name



Name of the app’s launcher script\.



*Type:*
string



## environment\.hakurei\.apps\.\<name>\.net



Whether to enable network access\.



*Type:*
boolean



*Default:*
` true `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.nix



Whether to enable nix daemon access\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.path



Custom executable path\.
Setting this to null will default to the start script\.



*Type:*
null or string



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.script



Application launch script\.



*Type:*
null or string



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.share



Package containing share files\.
Setting this to null will default package name to wrapper name\.



*Type:*
null or package



*Default:*
` null `



## environment\.hakurei\.apps\.\<name>\.shareUid



Whether to enable sharing identity with another application\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.tty



Whether to enable access to the controlling terminal\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.useCommonPaths



Whether to enable common extra paths\.



*Type:*
boolean



*Default:*
` true `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.userns



Whether to enable user namespace creation\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.verbose



Whether to enable launchers with verbose output\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.commonPaths



Common extra paths to make available to the container\.



*Type:*
list of (submodule)



*Default:*
` [ ] `



## environment\.hakurei\.commonPaths\.\*\.dev



Whether to enable use of device files\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.commonPaths\.\*\.dst



Mount point in container, same as src if null\.



*Type:*
null or string



*Default:*
` null `



## environment\.hakurei\.commonPaths\.\*\.require



Whether to enable start failure if the bind mount cannot be established for any reason\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.commonPaths\.\*\.src



Host filesystem path to make available to the container\.



*Type:*
string



## environment\.hakurei\.commonPaths\.\*\.write



Whether to enable mounting path as writable\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.extraHomeConfig



Extra home-manager configuration to merge with all target users\.



*Type:*
anything



## environment\.hakurei\.hsuPackage



The hsu package to use\.



*Type:*
package



*Default:*
` <derivation hakurei-hsu-0.0.2> `



## environment\.hakurei\.stateDir



The state directory where app home directories are stored\.



*Type:*
string



## environment\.hakurei\.users



Users allowed to spawn hakurei apps and their corresponding hakurei identity\.



*Type:*
attribute set of integer between 0 and 99 (both inclusive)


