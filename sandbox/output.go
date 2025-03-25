package sandbox

var msg Msg = new(DefaultMsg)

func GetOutput() Msg { return msg }
func SetOutput(v Msg) {
	if v == nil {
		msg = new(DefaultMsg)
	} else {
		msg = v
	}
}

func wrapErrSuffix(err error, a ...any) error {
	if err == nil {
		return nil
	}
	return msg.WrapErr(err, append(a, err)...)
}

func wrapErrSelf(err error) error {
	if err == nil {
		return nil
	}
	return msg.WrapErr(err, err.Error())
}
