package container

var msg Msg = new(DefaultMsg)

func GetOutput() Msg { return msg }
func SetOutput(v Msg) {
	if v == nil {
		msg = new(DefaultMsg)
	} else {
		msg = v
	}
}
