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
` <derivation hakurei-static-x86_64-unknown-linux-musl-0.3.1> `



## environment\.hakurei\.apps

Declaratively configured hakurei apps\.



*Type:*
attribute set of (submodule)



*Default:*
` { } `



## environment\.hakurei\.apps\.\<name>\.enablements\.dbus



Whether to proxy D-Bus\.



*Type:*
null or boolean



*Default:*
` true `



## environment\.hakurei\.apps\.\<name>\.enablements\.pulse



Whether to share the PulseAudio socket and cookie\.



*Type:*
null or boolean



*Default:*
` true `



## environment\.hakurei\.apps\.\<name>\.enablements\.wayland



Whether to share the Wayland socket\.



*Type:*
null or boolean



*Default:*
` true `



## environment\.hakurei\.apps\.\<name>\.enablements\.x11



Whether to share the X11 socket and allow connection\.



*Type:*
null or boolean



*Default:*
` false `



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
list of attribute set of anything



*Default:*
` [ ] `



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



## environment\.hakurei\.apps\.\<name>\.hostAbstract



Whether to enable share abstract unix socket scope\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.hostNet



Whether to enable share host net namespace\.



*Type:*
boolean



*Default:*
` true `



*Example:*
` true `



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



## environment\.hakurei\.apps\.\<name>\.shareRuntime



Whether to enable sharing of XDG_RUNTIME_DIR between containers under the same identity\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



## environment\.hakurei\.apps\.\<name>\.shareTmpdir



Whether to enable sharing of TMPDIR between containers under the same identity\.



*Type:*
boolean



*Default:*
` false `



*Example:*
` true `



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



## environment\.hakurei\.apps\.\<name>\.wait_delay



Duration to wait for after interrupting a container’s initial process in nanoseconds\.
A negative value causes the container to be terminated immediately on cancellation\.
Setting this to null defaults to five seconds\.



*Type:*
null or signed integer



*Default:*
` null `



## environment\.hakurei\.commonPaths



Common extra paths to make available to the container\.



*Type:*
list of attribute set of anything



*Default:*
` [ ] `



## environment\.hakurei\.extraHomeConfig



Extra home-manager configuration to merge with all target users\.



*Type:*
anything



## environment\.hakurei\.hsuPackage



The hsu package to use\.



*Type:*
package



*Default:*
` <derivation hakurei-hsu-0.3.1> `



## environment\.hakurei\.shell



Absolute path to preferred shell\.



*Type:*
string



*Default:*
` "/run/current-system/sw/bin/bash" `



## environment\.hakurei\.stateDir



The state directory where app home directories are stored\.



*Type:*
string



## environment\.hakurei\.users



Users allowed to spawn hakurei apps and their corresponding hakurei identity\.



*Type:*
attribute set of integer between 0 and 99 (both inclusive)


