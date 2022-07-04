package opt

import (
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"reflect"
)

const nullString = `null`

// Option is a nullable value compatible with sql/database, encoding/json and encoding/xml (so lang as the type parameter is compatible).
type Option[T any] struct {
	value T
	valid bool
}

// From returns a non-null Option
func From[T any](v T) Option[T] {
	return Option[T]{v, true}
}

// FromPtr returns a null Option for a nil pointer, otherwise a non-null Option
func FromPtr[T any](v *T) Option[T] {
	if v == nil {
		return Option[T]{}
	}

	return Option[T]{*v, true}
}

// Null returns true if the value is null
func (o Option[T]) Null() bool {
	return !o.valid
}

// Zero returns true if the value is null or if the contained value has its zero value
func (o Option[T]) NullOrZero() bool {
	if !o.valid {
		return true
	}

	v := reflect.ValueOf(o.value)
	return v.Kind() == reflect.Invalid || v.IsZero()
}

// Get returns the value of the contained type. If it is null, the zero value is returned.
func (o Option[T]) Get() T {
	return o.value
}

// String implements fmt.Stringer
func (o Option[T]) String() string {
	if !o.valid {
		return nullString
	}

	return fmt.Sprint(o.value)
}

// GoString implements fmt.GoStringer
func (o Option[T]) GoString() string {
	if !o.valid {
		return fmt.Sprintf(`option.Option[%s]{}`, reflect.TypeOf(*new(T)).String())
	}

	return fmt.Sprintf(`option.From[%s](%#v)`, reflect.TypeOf(*new(T)).String(), o.value)
}

// Value implements database/sql/driver.Valuer
func (o Option[T]) Value() (driver.Value, error) {
	if !o.valid {
		return nil, nil
	}

	return driver.DefaultParameterConverter.ConvertValue(o.value)
}

// Scan implements database/sql.Scanner
func (o *Option[T]) Scan(value any) error {
	if value == nil {
		*o = Option[T]{}
		return nil
	}

	o.valid = true
	return sqlConvertAssignRows(&o.value, value)
}

// MarshalJSON implements encoding/json.Marshaler
func (o Option[T]) MarshalJSON() ([]byte, error) {
	if !o.valid {
		return []byte(nullString), nil
	}

	return json.Marshal(o.value)
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (o *Option[T]) UnmarshalJSON(data []byte) error {
	if isNullString(string(data)) {
		*o = Option[T]{}
		return nil
	}

	o.valid = true
	return json.Unmarshal(data, &o.value)
}

// MarshalXML implements encoding/xml.Marshaler
func (o Option[T]) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if !o.valid {
		return nil
	}

	return e.EncodeElement(o.value, start)
}

// UnmarshalXML implements encoding/xml.Unmarshaler
func (o *Option[T]) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var ptr *T
	if err := d.DecodeElement(&ptr, &start); err != nil {
		return err
	}

	if ptr == nil {
		*o = Option[T]{}
		return nil
	}

	o.valid = true
	o.value = *ptr

	return nil
}

func isNullString(str string) bool {
	return str == `` || str == `null` || str == `NULL`
}
