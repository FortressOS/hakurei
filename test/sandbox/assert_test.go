package sandbox

func ReplaceFatal(f func(format string, v ...any)) { fatalfFunc = f }
