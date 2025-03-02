package sandbox

type F func(format string, v ...any)

func SwapPrint(f F) (old F) { old = printfFunc; printfFunc = f; return }
func SwapFatal(f F) (old F) { old = fatalfFunc; fatalfFunc = f; return }
