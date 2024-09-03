package app

func (a *App) Command() []string {
	return a.command
}

func (a *App) UID() int {
	return a.uid
}

func (a *App) AppendEnv(k, v string) {
	a.env = append(a.env, k+"="+v)
}
