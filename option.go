package opt

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var (
	_ json.Marshaler   = Option[struct{}]{}
	_ json.Unmarshaler = &Option[struct{}]{}
	_ driver.Valuer    = Option[struct{}]{}
	_ sql.Scanner      = &Option[struct{}]{}
)

// Option is a generic wrapper for optional values compatible with `encoding/json` and `database/sql`
type Option[T any] struct {
	V     T
	Valid bool
}

// New creates a new null Option[T]
func New[T any]() Option[T] {
	return Option[T]{}
}

// From creates a new non-null Option[T] with v
func From[T any](v T) Option[T] {
	return Option[T]{
		Valid: true,
		V:     v,
	}
}

// FromPtr creates an Option[T] that is null if v == nil,
// or non-null if v != nil,
// with the value pointed at by v
func FromPtr[T any](v *T) Option[T] {
	if v == nil {
		return Option[T]{}
	}

	return Option[T]{
		Valid: true,
		V:     *v,
	}
}

// Ptr returns a pointer to a copy of the value contained by Option.
func (o Option[T]) Ptr() *T {
	if !o.Valid {
		return nil
	}

	v := o.V
	return &v
}

// String implements fmt.Stringer
func (o Option[T]) String() string {
	if !o.Valid {
		return "null"
	}

	return fmt.Sprint(o.V)
}

// GoString implements fmt.GoStringer
func (o Option[T]) GoString() string {
	if !o.Valid {
		return fmt.Sprintf("opt.New[%s]()", getTypeName(reflect.TypeOf(&o.V).Elem()))
	}

	// for interfaces we need to explicitly mention the type since it cannot be inferred
	if t := reflect.TypeOf(&o.V).Elem(); t != nil && t.Kind() == reflect.Interface {
		return fmt.Sprintf("opt.From[%s](%#v)", getTypeName(t), o.V)
	}

	return fmt.Sprintf("opt.From(%#v)", o.V)
}

func getTypeName(t reflect.Type) string {
	name := t.Name()
	if name == "" {
		return fmt.Sprintf("%T", reflect.New(t).Elem().Interface())
	}

	path := t.PkgPath()
	i := strings.LastIndex(path, "/")
	if i == -1 {
		return name
	}

	return path[i+1:] + "." + name
}

// IsNull returns true if the value is null.
// It is meant to help improve readability.
func (o Option[T]) IsNull() bool {
	return !o.Valid
}

// MarshalJSON implements json.Marshaler
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.Valid {
		return []byte("null"), nil
	}

	return json.Marshal(o.V)
}

// UnmarshalJSON implements json.Unmarshaler
func (o *Option[T]) UnmarshalJSON(data []byte) error {
	*o = Option[T]{}

	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil
	}

	o.Valid = true

	err := json.Unmarshal(data, &o.V)
	if err != nil {
		return err
	}

	return nil
}

// Value implements driver.Valuer
func (o Option[T]) Value() (driver.Value, error) {
	if !o.Valid {
		return nil, nil
	}

	return driver.DefaultParameterConverter.ConvertValue(o.V)
}

// Scan implements sql.Scanner
func (o *Option[T]) Scan(data any) error {
	*o = Option[T]{}

	if data == nil {
		return nil
	}

	o.Valid = true
	err := scanAssign(&o.V, data)
	if err != nil {
		return err
	}

	return nil
}

// scanAssign is a copy of database/sql.assignConvertRows, with the following changes
//   - rows argument removed and any logic associated with it
//   - switch cases for sql.RawBytes removed
//   - nil checks removed, since we never pass a nil pointer
func scanAssign(dest, src any) error {
	// Common cases, without reflect.
	switch s := src.(type) {
	case string:
		switch d := dest.(type) {
		case *string:
			*d = s
			return nil
		case *[]byte:
			*d = []byte(s)
			return nil
		}
	case []byte:
		switch d := dest.(type) {
		case *string:
			*d = string(s)
			return nil
		case *any:
			*d = bytes.Clone(s)
			return nil
		case *[]byte:
			*d = bytes.Clone(s)
			return nil
		}
	case time.Time:
		switch d := dest.(type) {
		case *time.Time:
			*d = s
			return nil
		case *string:
			*d = s.Format(time.RFC3339Nano)
			return nil
		case *[]byte:
			*d = []byte(s.Format(time.RFC3339Nano))
			return nil
		}
	}

	var sv reflect.Value

	switch d := dest.(type) {
	case *string:
		sv = reflect.ValueOf(src)
		switch sv.Kind() {
		case reflect.Bool,
			reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			*d = asString(src)
			return nil
		}
	case *[]byte:
		sv = reflect.ValueOf(src)
		if b, ok := asBytes(nil, sv); ok {
			*d = b
			return nil
		}
	case *bool:
		bv, err := driver.Bool.ConvertValue(src)
		if err == nil {
			*d = bv.(bool)
		}
		return err
	case *any:
		*d = src
		return nil
	}

	if scanner, ok := dest.(sql.Scanner); ok {
		return scanner.Scan(src)
	}

	dpv := reflect.ValueOf(dest)
	if dpv.Kind() != reflect.Pointer {
		return errors.New("destination not a pointer")
	}
	if dpv.IsNil() {
		return errors.New("destination pointer is nil")
	}

	if !sv.IsValid() {
		sv = reflect.ValueOf(src)
	}

	dv := reflect.Indirect(dpv)
	if sv.IsValid() && sv.Type().AssignableTo(dv.Type()) {
		switch b := src.(type) {
		case []byte:
			dv.Set(reflect.ValueOf(bytes.Clone(b)))
		default:
			dv.Set(sv)
		}
		return nil
	}

	if dv.Kind() == sv.Kind() && sv.Type().ConvertibleTo(dv.Type()) {
		dv.Set(sv.Convert(dv.Type()))
		return nil
	}

	// The following conversions use a string value as an intermediate representation
	// to convert between various numeric types.
	//
	// This also allows scanning into user defined types such as "type Int int64".
	// For symmetry, also check for string destination types.
	switch dv.Kind() {
	case reflect.Pointer:
		if src == nil {
			dv.Set(reflect.Zero(dv.Type()))
			return nil
		}
		dv.Set(reflect.New(dv.Type().Elem()))
		return scanAssign(dv.Interface(), src)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if src == nil {
			return fmt.Errorf("converting NULL to %s is unsupported", dv.Kind())
		}
		s := asString(src)
		i64, err := strconv.ParseInt(s, 10, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetInt(i64)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if src == nil {
			return fmt.Errorf("converting NULL to %s is unsupported", dv.Kind())
		}
		s := asString(src)
		u64, err := strconv.ParseUint(s, 10, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetUint(u64)
		return nil
	case reflect.Float32, reflect.Float64:
		if src == nil {
			return fmt.Errorf("converting NULL to %s is unsupported", dv.Kind())
		}
		s := asString(src)
		f64, err := strconv.ParseFloat(s, dv.Type().Bits())
		if err != nil {
			err = strconvErr(err)
			return fmt.Errorf("converting driver.Value type %T (%q) to a %s: %v", src, s, dv.Kind(), err)
		}
		dv.SetFloat(f64)
		return nil
	case reflect.String:
		if src == nil {
			return fmt.Errorf("converting NULL to %s is unsupported", dv.Kind())
		}
		switch v := src.(type) {
		case string:
			dv.SetString(v)
			return nil
		case []byte:
			dv.SetString(string(v))
			return nil
		}
	}

	return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type %T", src, dest)
}

// scanAssign is a copy of database/sql.asString
func asString(src any) string {
	switch v := src.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	rv := reflect.ValueOf(src)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(rv.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(rv.Uint(), 10)
	case reflect.Float64:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 64)
	case reflect.Float32:
		return strconv.FormatFloat(rv.Float(), 'g', -1, 32)
	case reflect.Bool:
		return strconv.FormatBool(rv.Bool())
	}
	return fmt.Sprintf("%v", src)
}

// scanAssign is a copy of database/sql.asBytes
func asBytes(buf []byte, rv reflect.Value) (b []byte, ok bool) {
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(buf, rv.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.AppendUint(buf, rv.Uint(), 10), true
	case reflect.Float32:
		return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 32), true
	case reflect.Float64:
		return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 64), true
	case reflect.Bool:
		return strconv.AppendBool(buf, rv.Bool()), true
	case reflect.String:
		s := rv.String()
		return append(buf, s...), true
	}
	return
}

// scanAssign is a copy of database/sql.strconvErr
func strconvErr(err error) error {
	if ne, ok := err.(*strconv.NumError); ok {
		return ne.Err
	}
	return err
}
