package vtype

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
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

// OnSetFunc is an implementation of [OnSetter].
type OnSetFunc func(string) error

// OnSet calls the receiver function.
func (f OnSetFunc) OnSet(val string) error {
	return f(val)
}

// IsBool indicates whether the receiver function is intended to handle bool
// values.
func (f OnSetFunc) IsBool() bool { return false }

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

type Map struct {
	vals *map[any]any
}

// ConvertCompatible wraps compatible types.
func ConvertCompatible(val any) any {
	switch v := val.(type) {
	case func(string) error:
		return OnSetFunc(v)

	case func(bool) error:
		return OnSetBoolFunc(v)

	default:
		vo := reflect.ValueOf(val)
		if vo.Kind() == reflect.Ptr {
			vo = vo.Elem()
		}

		switch vo.Kind() {
		case reflect.Slice:
			s := MakeSlice(val)
			return &s
		}
	}

	return val
}

// Hydrate will parse the raw string value and use the result to update val.
// Valid val type values are:
//   - builtin: *string, *bool, error, *int, *int8, *int16, *int32, *int64,
//     *uint, *uint8, *uint16, *uint32, *uint64, *float32, *float64
//   - stdlib: *[time.Duration], [flag.Value]
//   - vtype: [TextMarshalUnmarshaler], [OnSetter], [StringSetter],
//     [OnSetFunc], [OnSetBoolFunc]
func Hydrate(val any, raw string) error {
	wrap := func(err error) error {
		return NewError(NewHydrateError(err, val))
	}

	switch v := val.(type) {
	case error:
		return wrap(v)

	case *string:
		*v = raw

	case *bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return wrap(err)
		}
		*v = b

	case *int:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return wrap(err)
		}
		*v = n

	case *int64:
		n, err := strconv.ParseInt(raw, 10, 0)
		if err != nil {
			return wrap(err)
		}
		*v = n

	case *int8:
		n, err := strconv.ParseInt(raw, 10, 8)
		if err != nil {
			return wrap(err)
		}
		*v = int8(n)

	case *int16:
		n, err := strconv.ParseInt(raw, 10, 16)
		if err != nil {
			return wrap(err)
		}
		*v = int16(n)

	case *int32:
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return wrap(err)
		}
		*v = int32(n)

	case *uint:
		n, err := strconv.ParseUint(raw, 10, 0)
		if err != nil {
			return wrap(err)
		}
		*v = uint(n)

	case *uint64:
		n, err := strconv.ParseUint(raw, 10, 0)
		if err != nil {
			return wrap(err)
		}
		*v = n

	case *uint8:
		n, err := strconv.ParseUint(raw, 10, 8)
		if err != nil {
			return wrap(err)
		}
		*v = uint8(n)

	case *uint16:
		n, err := strconv.ParseUint(raw, 10, 16)
		if err != nil {
			return wrap(err)
		}
		*v = uint16(n)

	case *uint32:
		n, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return wrap(err)
		}
		*v = uint32(n)

	case *float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return wrap(err)
		}
		*v = f

	case *float32:
		f, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return wrap(err)
		}
		*v = float32(f)

	case *time.Duration:
		d, err := time.ParseDuration(raw)
		if err != nil {
			return wrap(err)
		}
		*v = d

	case TextMarshalUnmarshaler:
		if err := v.UnmarshalText([]byte(raw)); err != nil {
			return wrap(err)
		}

	case StringSetter:
		if err := v.Set(raw); err != nil {
			return wrap(err)
		}

	case OnSetter:
		if err := v.OnSet(raw); err != nil {
			return wrap(err)
		}

	default:
		return wrap(ErrUnsupportedType)
	}

	return nil
}

type ValueTypeNamer interface {
	ValueTypeName() string
}

// ValueTypeName returns a "best effort" text representation of the value's
// type name. Explicit values are communicated by types implementing
// [ValueTypeNamer].
func ValueTypeName(val any) string {
	switch v := val.(type) {
	case ValueTypeNamer:
		return v.ValueTypeName()

	case interface{ IsBool() bool }:
		if v.IsBool() {
			return "bool"
		}
		return "value"

	case TextMarshalUnmarshaler, StringSetter:
		return "value"

	case error:
		return ""

	default:
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		return rv.Type().Name()
	}
}

type DefaultValueTexter interface {
	DefaultValueText() string
}

// DefaultValueText returns a "best effort" text representation of the value.
// Explicit values are communicated by types implementing [DefaultValueTexter].
func DefaultValueText(val any) string {
	switch v := val.(type) {
	case DefaultValueTexter:
		return v.DefaultValueText()

	case TextMarshalUnmarshaler:
		t, err := v.MarshalText()
		if err != nil {
			return err.Error()
		}
		return string(t)

	case error:
		return ""

	case fmt.Stringer:
		return v.String()

	default:
		if reflect.ValueOf(val).Kind() == reflect.Func {
			return ""
		}

		vo := reflect.ValueOf(val)
		if vo.Kind() == reflect.Ptr {
			vo = vo.Elem()
		}
		return fmt.Sprint(vo)
	}
}
