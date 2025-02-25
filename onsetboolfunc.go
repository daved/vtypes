package vtypes

import "strconv"

// OnSetBoolFunc is an implementation of [OnSetter].
type OnSetBoolFunc func(bool) error

// OnSet calls the receiver function, first parsing the string value as a bool
// type.
func (f OnSetBoolFunc) OnSet(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	return f(b)
}

// IsBool indicates whether the receiver function is intended to handle bool
// values.
func (f OnSetBoolFunc) IsBool() bool { return true }
