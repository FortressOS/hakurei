package seccomp_test

import (
	. "hakurei.app/container/comp"
	. "hakurei.app/container/seccomp"
)

var bpfExpected = bpfLookup{
	{AllowMultiarch | AllowCAN |
		AllowBluetooth, PresetExt |
		PresetDenyNS | PresetDenyTTY | PresetDenyDevel |
		PresetLinux32}: toHash(
		"e99dd345e195413473d3cbee07b4ed57b908bfa89ea2072fe93482847f50b5b758da17e74ca2bbc00813de49a2b9bf834c024ed48850be69b68a9a4c5f53a9db"),

	{0, 0}: toHash(
		"95ec69d017733e072160e0da80fdebecdf27ae8166f5e2a731270c98ea2d2946cb5231029063668af215879155da21aca79b070e04c0ee9acdf58f55cfa815a5"),
	{0, PresetExt}: toHash(
		"dc7f2e1c5e829b79ebb7efc759150f54a83a75c8df6fee4dce5dadc4736c585d4deebfeb3c7969af3a077e90b77bb4741db05d90997c8659b95891206ac9952d"),
	{0, PresetStrict}: toHash(
		"e880298df2bd6751d0040fc21bc0ed4c00f95dc0d7ba506c244d8b8cf6866dba8ef4a33296f287b66cccc1d78e97026597f84cc7dec1573e148960fbd35cd735"),
	{0, PresetDenyNS | PresetDenyTTY | PresetDenyDevel}: toHash(
		"39871b93ffafc8b979fcedc0b0c37b9e03922f5b02748dc5c3c17c92527f6e022ede1f48bff59246ea452c0d1de54827808b1a6f84f32bbde1aa02ae30eedcfa"),
	{0, PresetExt | PresetDenyDevel}: toHash(
		"c698b081ff957afe17a6d94374537d37f2a63f6f9dd75da7546542407a9e32476ebda3312ba7785d7f618542bcfaf27ca27dcc2dddba852069d28bcfe8cad39a"),
	{0, PresetExt | PresetDenyNS | PresetDenyDevel}: toHash(
		"0b76007476c1c9e25dbf674c29fdf609a1656a70063e49327654e1b5360ad3da06e1a3e32bf80e961c5516ad83d4b9e7e9bde876a93797e27627d2555c25858b"),
}
