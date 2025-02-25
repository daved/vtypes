package vtypes

// OnSetFunc is an implementation of [OnSetter].
type OnSetFunc func(string) error

// OnSet calls the receiver function.
func (f OnSetFunc) OnSet(val string) error {
	return f(val)
}

// IsBool indicates whether the receiver function is intended to handle bool
// values.
func (f OnSetFunc) IsBool() bool { return false }
