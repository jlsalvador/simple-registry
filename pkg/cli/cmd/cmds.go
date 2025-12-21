package cmd

type Cmd struct {
	Name string
	Help string
	Fn   func() error
}
