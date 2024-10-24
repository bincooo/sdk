package sdk

type Context struct {
	In,
	Out []any
	Name     string
	Receiver any
	Do       func()
}
