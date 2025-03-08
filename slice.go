package vtypes

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Slice is an implementation of TextMarshalUnmarshaler that wraps a slice value
// (possibly with multiple levels of pointers) of any type supported by [Hydrate].
// Behavior can be configured to treat each UnmarshalText call as a set of values.
// The underlying slice is only initialized if values are added; otherwise, nil
// pointers in the chain remain nil.
type Slice struct {
	ptrValue any // Stores the original value (e.g., **[]int, *[]int, []int)
	started  bool

	TypeName  string
	SplitEach bool
	Separator string
	NonAccum  bool
}

// MakeSlice returns an instance of Slice.
func MakeSlice(ptrValue any) Slice {
	return Slice{
		ptrValue:  ptrValue,
		Separator: ",", // Default separator for comma-separated lists
	}
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (s *Slice) UnmarshalText(text []byte) error {
	// Preserve nil state if no text
	if len(text) == 0 {
		return nil
	}

	// Get the value and determine its indirection level
	v := reflect.ValueOf(s.ptrValue)
	pointerLevels := 0
	for v.Kind() == reflect.Pointer {
		pointerLevels++
		if v.IsNil() {
			// Initialize only if we have values to add
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return errors.New("slice: contained value is not a slice or pointer to a slice")
	}

	// Initialize or reset only if necessary
	if !s.started || s.NonAccum {
		slice := reflect.MakeSlice(v.Type(), 0, 0)
		s.setValue(slice)
	}
	s.started = true

	valType := v.Type().Elem()
	sep := s.Separator
	if !s.SplitEach {
		// Only use a different separator if explicitly intended; otherwise, keep default
		sep = s.Separator // Default to "," unless overridden
	}

	for _, chunk := range bytes.Split(text, []byte(sep)) {
		if len(chunk) == 0 {
			continue // Skip empty chunks
		}
		item := reflect.New(valType)
		if err := Hydrate(item.Interface(), string(chunk)); err != nil {
			return fmt.Errorf("slice: unmarshal text: %w", err)
		}
		slice := reflect.Append(v, item.Elem())
		s.setValue(slice)
	}

	return nil
}

// MarshalText implements [encoding.TextMarshaler].
func (s *Slice) MarshalText() ([]byte, error) {
	v := reflect.ValueOf(s.ptrValue)
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil // Return nil text for nil pointers
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		return nil, errors.New("slice: contained value is not a slice or pointer to a slice")
	}

	out := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		out[i] = fmt.Sprint(v.Index(i).Interface())
	}
	return []byte(strings.Join(out, s.Separator)), nil
}

// ValueTypeName returns the name of the underlying slice element type, adding
// information if unmarshaling is configured to handle a set of values.
func (s *Slice) ValueTypeName() string {
	rv := reflect.ValueOf(s.ptrValue)
	for rv.Kind() == reflect.Pointer {
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
	rv := reflect.ValueOf(s.ptrValue)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	return rv.Type().Elem().Kind() == reflect.Bool
}

// Value returns the original value with its pointer chain.
func (s *Slice) Value() any {
	return s.ptrValue
}

// setValue updates the slice value through the pointer chain.
func (s *Slice) setValue(slice reflect.Value) {
	v := reflect.ValueOf(s.ptrValue)
	for i := 0; i < s.pointerLevels()-1; i++ {
		v = v.Elem()
	}
	v.Elem().Set(slice)
}

// pointerLevels returns the number of pointer levels in ptrValue.
func (s *Slice) pointerLevels() int {
	levels := 0
	v := reflect.ValueOf(s.ptrValue)
	for v.Kind() == reflect.Pointer {
		levels++
		v = v.Elem()
	}
	return levels
}
