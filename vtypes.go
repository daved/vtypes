// Package vtypes provides value type handling helpers and types.
package vtypes

import (
	"fmt"
	"reflect"
	"strconv"
	"time"
)

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
//   - vtypes: [TextMarshalUnmarshaler], [OnSetter], [StringSetter],
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
		return wrap(ErrTypeUnsupported)
	}

	return nil
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
