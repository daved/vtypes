package vtypes

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Slice is an implementation of TextMarshalUnmarshaler that wraps a pointer to
// a slice of any type supported by [Hydrate]. Behavior can be configured to
// treat each UnmarshalText call as a set of values. The first call to
// UnmarshalText will clear the underlying slice, and subsequent calls are
// accumulative unless otherwise configured.
type Slice struct {
	ptrToSlice any
	started    bool

	TypeName  string
	SplitEach bool
	Separator string
	NonAccum  bool
}

// MakeSlice returns an instance of Slice.
func MakeSlice(ptrToSlice any) Slice {
	return Slice{
		ptrToSlice: ptrToSlice,
		Separator:  ",",
	}
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (s *Slice) UnmarshalText(text []byte) error {
	vo := reflect.ValueOf(s.ptrToSlice)
	isPtr := vo.Kind() == reflect.Pointer
	if isPtr {
		vo = vo.Elem()
	}
	if !isPtr || vo.Kind() != reflect.Slice {
		return errors.New("slice: contained value is not a pointer to a slice")
	}

	if !s.started || s.NonAccum {
		slice := reflect.MakeSlice(vo.Type(), 0, 0)
		reflect.ValueOf(s.ptrToSlice).Elem().Set(slice)
	}
	s.started = true

	valType := vo.Type().Elem()

	sep := s.Separator
	if !s.SplitEach {
		sep = "<><>"
	}

	for _, chunk := range bytes.Split(text, []byte(sep)) {
		item := reflect.New(valType)
		if err := Hydrate(item.Interface(), string(chunk)); err != nil {
			return fmt.Errorf("slice: unmarshal text: %w", err)
		}

		slice := reflect.Append(vo, item.Elem())
		reflect.ValueOf(s.ptrToSlice).Elem().Set(slice)
	}

	return nil
}

// MarshalText implements [encoding.TextMarshaler].
func (s *Slice) MarshalText() ([]byte, error) {
	vo := reflect.ValueOf(s.ptrToSlice)
	isPtr := vo.Kind() == reflect.Pointer
	if isPtr {
		vo = vo.Elem()
	}
	if !isPtr || vo.Kind() != reflect.Slice {
		return nil, errors.New("slice: contained value is not a pointer to a slice")
	}

	out := make([]string, vo.Len())
	for i := 0; i < vo.Len(); i++ {
		out[i] = fmt.Sprint(vo.Index(i).Interface())
	}
	return []byte(strings.Join(out, s.Separator)), nil
}

// ValueTypeName returns the name of the underlying slice element type, adding
// information if unmarshaling is configured to handle a set of values.
func (s *Slice) ValueTypeName() string {
	rv := reflect.ValueOf(s.ptrToSlice)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	name := rv.Type().Elem().Name()

	if s.SplitEach {
		name += fmt.Sprintf("(multisep:%s)", s.Separator)
	}

	return name
}

// IsBool indicates whether the underlying slice element type is bool.
func (s *Slice) IsBool() bool {
	return reflect.ValueOf(s.ptrToSlice).Elem().Type().Elem().Kind() == reflect.Bool
}
