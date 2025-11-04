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
		"1431c013f2ddac3adae577821cb5d351b1514e7c754d62346ddffd31f46ea02fb368e46e3f8104f81019617e721fe687ddd83f1e79580622ccc991da12622170"),

	{0, 0}: toHash(
		"450c21210dbf124dfa7ae56d0130f9c2e24b26f5bce8795ee75766c75850438ff9e7d91c5e73d63bbe51a5d4b06c2a0791c4de2903b2b9805f16265318183235"),
	{0, PresetExt}: toHash(
		"d971d0f2d30f54ac920fc6d84df2be279e9fd28cf2d48be775d7fdbd790b750e1369401cd3bb8bcf9ba3adb91874fe9792d9e3f62209b8ee59c9fdd2ddd10c7b"),
	{0, PresetStrict}: toHash(
		"79318538a3dc851314b6bd96f10d5861acb2aa7e13cb8de0619d0f6a76709d67f01ef3fd67e195862b02f9711e5b769bc4d1eb4fc0dfc41a723c89c968a93297"),
	{0, PresetDenyNS | PresetDenyTTY | PresetDenyDevel}: toHash(
		"228286c2f5df8e44463be0a57b91977b7f38b63b09e5d98dfabe5c61545b8f9ac3e5ea3d86df55d7edf2ce61875f0a5a85c0ab82800bef178c42533e8bdc9a6c"),
	{0, PresetExt | PresetDenyDevel}: toHash(
		"433ce9b911282d6dcc8029319fb79b816b60d5a795ec8fc94344dd027614d68f023166a91bb881faaeeedd26e3d89474e141e5a69a97e93b8984ca8f14999980"),
	{0, PresetExt | PresetDenyNS | PresetDenyDevel}: toHash(
		"cf1f4dc87436ba8ec95d268b663a6397bb0b4a5ac64d8557e6cc529d8b0f6f65dad3a92b62ed29d85eee9c6dde1267757a4d0f86032e8a45ca1bceadfa34cf5e"),
}
