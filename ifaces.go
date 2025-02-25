package vtypes

import (
	"encoding"
	"fmt"
)

// TextMarshalUnmarshaler descibes types that are able to be marshaled to and
// unmarshaled from text.
type TextMarshalUnmarshaler interface {
	encoding.TextUnmarshaler
	encoding.TextMarshaler
}

// OnSetter describes types that will have a callback (OnSet) run when
// hydrated. The IsBool method conveys type information used to determine the
// default text and can be used to inform external handlers.
type OnSetter interface {
	OnSet(val string) error
	IsBool() bool
}

// StringSetter describes types that are set by and expressed as a string value.
type StringSetter interface {
	Set(val string) error
	fmt.Stringer
}

type ValueTypeNamer interface {
	ValueTypeName() string
}

type DefaultValueTexter interface {
	DefaultValueText() string
}
