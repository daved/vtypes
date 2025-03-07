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
		if vo.Kind() == reflect.Pointer {
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

	tmpVal, pointerChain, err := tempValue(val)
	if err != nil {
		return wrap(err)
	}

	err = hydrateValue(tmpVal, raw)
	if err != nil {
		return wrap(err)
	}

	err = assignThroughChain(tmpVal, pointerChain)
	if err != nil {
		return wrap(err)
	}

	return nil
}

func tempValue(val any) (prepared any, pointerChain []reflect.Value, err error) {
	v := reflect.ValueOf(val)
	if v.Kind() != reflect.Pointer {
		return nil, nil, ErrTypeUnsupported
	}

	// Collect all pointer levels
	pointerChain = make([]reflect.Value, 0)
	current := v

	// Keep following pointers until we hit a non-pointer or nil at the deepest level
	for current.Kind() == reflect.Pointer {
		if current.IsNil() {
			// Create new values for nil pointers
			newVal := reflect.New(current.Type().Elem())
			current.Set(newVal)
		}
		pointerChain = append(pointerChain, current)
		current = current.Elem()
	}

	// We want to work with a single pointer to the final value
	if len(pointerChain) < 1 {
		return nil, nil, ErrTypeUnsupported
	}

	// The prepared value will be the last pointer in the chain
	prepared = pointerChain[len(pointerChain)-1].Interface()
	return prepared, pointerChain, nil
}

// hydrateValue handles the actual parsing and assignment to the prepared single-pointer value
func hydrateValue(val any, raw string) error {
	switch v := val.(type) {
	case error:
		return v

	case *string:
		*v = raw

	case *bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		*v = b

	case *int:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return err
		}
		*v = n

	case *int64:
		n, err := strconv.ParseInt(raw, 10, 0)
		if err != nil {
			return err
		}
		*v = n

	case *int8:
		n, err := strconv.ParseInt(raw, 10, 8)
		if err != nil {
			return err
		}
		*v = int8(n)

	case *int16:
		n, err := strconv.ParseInt(raw, 10, 16)
		if err != nil {
			return err
		}
		*v = int16(n)

	case *int32:
		n, err := strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return err
		}
		*v = int32(n)

	case *uint:
		n, err := strconv.ParseUint(raw, 10, 0)
		if err != nil {
			return err
		}
		*v = uint(n)

	case *uint64:
		n, err := strconv.ParseUint(raw, 10, 0)
		if err != nil {
			return err
		}
		*v = n

	case *uint8:
		n, err := strconv.ParseUint(raw, 10, 8)
		if err != nil {
			return err
		}
		*v = uint8(n)

	case *uint16:
		n, err := strconv.ParseUint(raw, 10, 16)
		if err != nil {
			return err
		}
		*v = uint16(n)

	case *uint32:
		n, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			return err
		}
		*v = uint32(n)

	case *float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}
		*v = f

	case *float32:
		f, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return err
		}
		*v = float32(f)

	case *time.Duration:
		d, err := time.ParseDuration(raw)
		if err != nil {
			return err
		}
		*v = d

	case TextMarshalUnmarshaler:
		if err := v.UnmarshalText([]byte(raw)); err != nil {
			return err
		}

	case StringSetter:
		if err := v.Set(raw); err != nil {
			return err
		}

	case OnSetter:
		if err := v.OnSet(raw); err != nil {
			return err
		}

	default:
		return ErrTypeUnsupported
	}

	return nil
}

// assignThroughChain propagates the value back through the pointer chain
func assignThroughChain(prepared any, pointerChain []reflect.Value) error {
	if len(pointerChain) == 0 {
		return nil
	}

	preparedVal := reflect.ValueOf(prepared)
	if preparedVal.Kind() != reflect.Pointer {
		return ErrTypeUnsupported
	}

	// Start with the actual value
	currentVal := preparedVal.Elem()

	// Work backwards through the pointer chain
	for i := len(pointerChain) - 1; i >= 0; i-- {
		pointer := pointerChain[i]
		pointer.Elem().Set(currentVal)
		currentVal = pointer
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

	case nil, error:
		return ""

	default:
		t := reflect.TypeOf(val)
		if t == nil {
			return ""
		}

		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		return t.Name()
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
		for vo.Kind() == reflect.Pointer {
			if vo.IsNil() {
				return ""
			}
			vo = vo.Elem()
		}
		return fmt.Sprint(vo)
	}
}
