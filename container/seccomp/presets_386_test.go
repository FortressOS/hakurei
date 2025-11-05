package seccomp_test

import (
	. "hakurei.app/container/seccomp"
	. "hakurei.app/container/std"
)

var bpfExpected = bpfLookup{
	{AllowMultiarch | AllowCAN |
		AllowBluetooth, PresetExt |
		PresetDenyNS | PresetDenyTTY | PresetDenyDevel |
		PresetLinux32}: toHash(
		"e67735d24caba42b6801e829ea4393727a36c5e37b8a51e5648e7886047e8454484ff06872aaef810799c29cbd0c1b361f423ad0ef518e33f68436372cc90eb1"),

	{0, 0}: toHash(
		"5dbcc08a4a1ccd8c12dd0cf6d9817ea6d4f40246e1db7a60e71a50111c4897d69f6fb6d710382d70c18910c2e4fa2d2aeb2daed835dd2fabe3f71def628ade59"),
	{0, PresetExt}: toHash(
		"d6c0f130dbb5c793d1c10f730455701875778138bd2d03ca009d674842fd97a10815a8c539b76b7801a73de19463938701216b756c053ec91cfe304cba04a0ed"),
	{0, PresetStrict}: toHash(
		"af7d7b66f2e83f9a850472170c1b83d1371426faa9d0dee4e85b179d3ec75ca92828cb8529eb3012b559497494b2eab4d4b140605e3a26c70dfdbe5efe33c105"),
	{0, PresetDenyNS | PresetDenyTTY | PresetDenyDevel}: toHash(
		"adfb4397e6eeae8c477d315d58204aae854d60071687b8df4c758e297780e02deee1af48328cef80e16e4d6ab1a66ef13e42247c3475cf447923f15cbc17a6a6"),
	{0, PresetExt | PresetDenyDevel}: toHash(
		"5d641321460cf54a7036a40a08e845082e1f6d65b9dee75db85ef179f2732f321b16aee2258b74273b04e0d24562e8b1e727930a7e787f41eb5c8aaa0bc22793"),
	{0, PresetExt | PresetDenyNS | PresetDenyDevel}: toHash(
		"b1f802d39de5897b1e4cb0e82a199f53df0a803ea88e2fd19491fb8c90387c9e2eaa7e323f565fecaa0202a579eb050531f22e6748e04cfd935b8faac35983ec"),
}
