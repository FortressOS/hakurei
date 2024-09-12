package system

const (
	xdgRuntimeDir = "XDG_RUNTIME_DIR"
)

type Values struct {
	Share   string
	Runtime string
	RunDir  string
}

var V *Values
