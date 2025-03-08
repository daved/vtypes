package vtypes_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/daved/vtypes"
)

func TestSurfaceHydrate(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		raw     string
		want    any
		wantErr bool
	}{
		// Single pointer tests
		{name: "string single", input: new(string), raw: "hello", want: ptr("hello")},
		{name: "bool single", input: new(bool), raw: "true", want: ptr(true)},
		{name: "int single", input: new(int), raw: "42", want: ptr(42)},
		{name: "uint single", input: new(uint), raw: "42", want: ptr(uint(42))},
		{name: "float64 single", input: new(float64), raw: "3.14", want: ptr(3.14)},
		{name: "duration single", input: new(time.Duration), raw: "1h", want: ptr(time.Hour)},

		// Double pointer tests
		{name: "string double", input: ptr(new(string)), raw: "hello", want: ptr(ptr("hello"))},
		{name: "bool double", input: ptr(new(bool)), raw: "false", want: ptr(ptr(false))},
		{name: "int double", input: ptr(new(int)), raw: "-42", want: ptr(ptr(-42))},
		{name: "uint double", input: ptr(new(uint)), raw: "42", want: ptr(ptr(uint(42)))},
		{name: "float32 double", input: ptr(new(float32)), raw: "3.14", want: ptr(ptr(float32(3.14)))},

		// Triple pointer tests
		{name: "string triple", input: ptr(ptr(new(string))), raw: "test", want: ptr(ptr(ptr("test")))},
		{name: "int triple", input: ptr(ptr(new(int))), raw: "123", want: ptr(ptr(ptr(123)))},

		// Error cases
		{name: "invalid bool", input: new(bool), raw: "notabool", wantErr: true},
		{name: "invalid int", input: new(int), raw: "notanint", wantErr: true},
		{name: "invalid float", input: new(float64), raw: "notafloat", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vtypes.Hydrate(tt.input, tt.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("Hydrate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Follow all pointers to get the final value
			got := reflect.ValueOf(tt.input)
			want := reflect.ValueOf(tt.want)
			for got.Kind() == reflect.Pointer && want.Kind() == reflect.Pointer {
				got = got.Elem()
				want = want.Elem()
			}

			if !reflect.DeepEqual(got.Interface(), want.Interface()) {
				t.Errorf("Hydrate() got = %v, want %v", got.Interface(), want.Interface())
			}
		})
	}
}

// ptr is a helper to create pointers of any type
func ptr[T any](v T) *T {
	return &v
}

func TestConvertCompatibleWithDoublePointerNilSlice(t *testing.T) {
	var slice *[]int // nil
	p1 := &slice     // **[]int where *[]int is nil

	result := vtypes.ConvertCompatible(p1)
	rv := reflect.ValueOf(result)
	if rv.Kind() != reflect.Pointer {
		t.Errorf("Expected pointer to Slice, got %v", rv.Kind())
	}
	rv = rv.Elem()
	if rv.Type() != reflect.TypeOf(vtypes.Slice{}) {
		t.Errorf("Expected Slice type, got %v", rv.Type())
	}

	sliceVal := rv.Interface().(vtypes.Slice)
	slicePtr := sliceVal.Value().(**[]int)
	if slicePtr == nil {
		t.Errorf("Expected non-nil **[]int pointer, got nil")
	}
	if *slicePtr != nil {
		t.Errorf("Expected nil *[]int pointer, got %v", *slicePtr)
	}

	err := sliceVal.UnmarshalText([]byte(""))
	if err != nil {
		t.Errorf("UnmarshalText error: %v", err)
	}
	slicePtr = sliceVal.Value().(**[]int)
	if *slicePtr != nil {
		t.Errorf("Expected nil *[]int pointer after empty unmarshal, got %v", *slicePtr)
	}

	err = sliceVal.UnmarshalText([]byte("1,2,3"))
	if err != nil {
		t.Errorf("UnmarshalText error: %v", err)
	}
	slicePtr = sliceVal.Value().(**[]int)
	if slicePtr == nil {
		t.Errorf("Expected non-nil **[]int pointer after unmarshal, got nil")
	}
	if *slicePtr == nil {
		t.Errorf("Expected non-nil *[]int pointer after unmarshal, got nil")
	}
	if !reflect.DeepEqual(**slicePtr, []int{1, 2, 3}) {
		t.Errorf("Expected slice values [1, 2, 3], got %v", **slicePtr)
	}
}

func TestConvertCompatibleWithTriplePointerSlice(t *testing.T) {
	slice := []int{1, 2, 3}
	p1 := &slice
	p2 := &p1
	p3 := &p2

	result := vtypes.ConvertCompatible(p3)
	rv := reflect.ValueOf(result)
	if rv.Kind() != reflect.Pointer {
		t.Errorf("Expected pointer to Slice, got %v", rv.Kind())
	}
	rv = rv.Elem()
	if rv.Type() != reflect.TypeOf(vtypes.Slice{}) {
		t.Errorf("Expected Slice type, got %v", rv.Type())
	}

	sliceVal := rv.Interface().(vtypes.Slice)
	slicePtr := sliceVal.Value().(***[]int)
	if slicePtr == nil {
		t.Errorf("Expected non-nil slice pointer, got nil")
	}
	if !reflect.DeepEqual(***slicePtr, []int{1, 2, 3}) {
		t.Errorf("Expected slice values [1, 2, 3], got %v", ***slicePtr)
	}
}

func TestConvertCompatibleWithSinglePointerSlice(t *testing.T) {
	slice := []int{1, 2, 3}
	p1 := &slice

	result := vtypes.ConvertCompatible(p1)
	rv := reflect.ValueOf(result)
	if rv.Kind() != reflect.Pointer {
		t.Errorf("Expected pointer to Slice, got %v", rv.Kind())
	}
	rv = rv.Elem()
	if rv.Type() != reflect.TypeOf(vtypes.Slice{}) {
		t.Errorf("Expected Slice type, got %v", rv.Type())
	}

	sliceVal := rv.Interface().(vtypes.Slice)
	slicePtr := sliceVal.Value().(*[]int)
	if slicePtr == nil {
		t.Errorf("Expected non-nil slice pointer, got nil")
	}
	if !reflect.DeepEqual(*slicePtr, []int{1, 2, 3}) {
		t.Errorf("Expected slice values [1, 2, 3], got %v", *slicePtr)
	}
}
