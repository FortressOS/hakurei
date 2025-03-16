package fmsg

type Output struct{}

func (Output) IsVerbose() bool                         { return Load() }
func (Output) Verbose(v ...any)                        { Verbose(v...) }
func (Output) Verbosef(format string, v ...any)        { Verbosef(format, v...) }
func (Output) WrapErr(err error, a ...any) error       { return WrapError(err, a...) }
func (Output) PrintBaseErr(err error, fallback string) { PrintBaseError(err, fallback) }
func (Output) Suspend()                                { Suspend() }
func (Output) Resume() bool                            { return Resume() }
func (Output) BeforeExit()                             { BeforeExit() }
