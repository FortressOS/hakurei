package fhs

import (
	_ "unsafe" // for go:linkname

	"hakurei.app/container/check"
)

/* constants in this file bypass abs check, be extremely careful when changing them! */

//go:linkname unsafeAbs hakurei.app/container/check.unsafeAbs
func unsafeAbs(_ string) *check.Absolute

var (
	// AbsRoot is [Root] as [check.Absolute].
	AbsRoot = unsafeAbs(Root)
	// AbsEtc is [Etc] as [check.Absolute].
	AbsEtc = unsafeAbs(Etc)
	// AbsTmp is [Tmp] as [check.Absolute].
	AbsTmp = unsafeAbs(Tmp)

	// AbsRun is [Run] as [check.Absolute].
	AbsRun = unsafeAbs(Run)
	// AbsRunUser is [RunUser] as [check.Absolute].
	AbsRunUser = unsafeAbs(RunUser)

	// AbsUsrBin is [UsrBin] as [check.Absolute].
	AbsUsrBin = unsafeAbs(UsrBin)

	// AbsVar is [Var] as [check.Absolute].
	AbsVar = unsafeAbs(Var)
	// AbsVarLib is [VarLib] as [check.Absolute].
	AbsVarLib = unsafeAbs(VarLib)

	// AbsDev is [Dev] as [check.Absolute].
	AbsDev = unsafeAbs(Dev)
	// AbsProc is [Proc] as [check.Absolute].
	AbsProc = unsafeAbs(Proc)
	// AbsSys is [Sys] as [check.Absolute].
	AbsSys = unsafeAbs(Sys)
)
