package opt_test

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/FallenTaters/opt"
)

func TestOption(t *testing.T) {
	t.Run("pointer methods", func(t *testing.T) {
		assertEq(t, opt.FromPtr[int](nil), opt.New[int]())
		assertEq(t, opt.FromPtr(ptr(0)), opt.From(0))
		assertEq(t, opt.FromPtr(ptr(3)), opt.From(3))

		assertEq(t, opt.New[int]().Ptr(), nil)
		assertEq(t, *opt.From(0).Ptr(), 0)
		assertEq(t, *opt.From(1).Ptr(), 1)
	})

	t.Run("IsNull", func(t *testing.T) {
		assertEq(t, opt.New[int]().IsNull(), true)
		assertEq(t, opt.From(0).IsNull(), false)
		assertEq(t, opt.From(1).IsNull(), false)
	})

}

func TestGoString(t *testing.T) {
	assertEq(t, opt.New[int]().GoString(), "opt.New[int]()")
	assertEq(t, opt.From(1).GoString(), "opt.From(1)")
	assertEq(t, opt.FromPtr[int](nil).GoString(), "opt.New[int]()")
	assertEq(t, opt.FromPtr[TestStruct1](nil).GoString(), "opt.New[TestStruct1]()")
	assertEq(t, opt.From(TestStruct1{"hello"}).GoString(), "opt.From(opt_test.TestStruct1{V:\"hello\"})")
	assertEq(t, opt.New[sql.NullInt64]().GoString(), "opt.New[sql.NullInt64]()")
	assertEq(t, opt.From(sql.NullInt64{}).GoString(), "opt.From(sql.NullInt64{Int64:0, Valid:false})")
	assertEq(t, opt.From(sql.NullInt64{Valid: true, Int64: 1}).GoString(), "opt.From(sql.NullInt64{Int64:1, Valid:true})")
	assertEq(t, opt.New[sql.Scanner]().GoString(), "opt.New[sql.Scanner]()")
	assertEq(t, opt.From[sql.Scanner](&sql.NullInt64{}).GoString(), "opt.From[sql.Scanner](&sql.NullInt64{Int64:0, Valid:false})")
	assertEq(t, opt.From[sql.Scanner](&sql.NullInt64{Valid: true, Int64: 1}).GoString(), "opt.From[sql.Scanner](&sql.NullInt64{Int64:1, Valid:true})")
	assertEq(t, opt.New[chan int]().GoString(), "opt.New[chan int]()")
	assertEq(t, opt.New[func()]().GoString(), "opt.New[func()]()")

	// assertEq(t, opt.From[sql.Scanner](nil).GoString(), "opt.From[sql.Scanner](<nil>)")
	// assertEq(t, opt.From(make(chan int)).GoString(), "opt.From((chan int)(0xc0001a4c60))")
}

func TestOptionInt64(t *testing.T) {
	t.Run("sql.Scanner", func(t *testing.T) {
		cases := []any{
			int64(0),
			int64(1),
			float64(0.0),
			float64(1.0),
			float64(1.1),
			true,
			false,
			[]byte(nil),
			[]byte{},
			[]byte("hello"),
			"",
			"hello",
			time.Now(),
			time.Time{},
			nil,
		}

		for _, v := range cases {
			t.Run(fmt.Sprint(v), func(t *testing.T) {
				var sqlInt sql.NullInt64
				var optInt opt.Option[int64]

				sqlErr := sqlInt.Scan(v)
				optErr := optInt.Scan(v)

				assertErrorEq(t, optErr, sqlErr)
				assertEq(t, optInt.Valid, sqlInt.Valid)
				assertEq(t, optInt.V, sqlInt.Int64)
			})
		}
	})

	t.Run("driver.Valuer", func(t *testing.T) {
		cases := []struct {
			sql    sql.NullInt64
			option opt.Option[int64]
		}{
			{
				sql:    sql.NullInt64{},
				option: opt.Option[int64]{},
			},
			{
				sql:    sql.NullInt64{Valid: true, Int64: 0},
				option: opt.From(int64(0)),
			},
			{
				sql:    sql.NullInt64{Valid: true, Int64: 1},
				option: opt.From(int64(1)),
			},
			{
				sql:    sql.NullInt64{Valid: true, Int64: -1},
				option: opt.From(int64(-1)),
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c.option), func(t *testing.T) {
				sqlVal, sqlErr := c.sql.Value()
				optVal, optErr := c.option.Value()

				assertErrorEq(t, optErr, sqlErr)
				assertEq(t, optVal, sqlVal)
			})
		}
	})

	t.Run("json.Marshaler", func(t *testing.T) {
		cases := []struct {
			ptr    *int64
			option opt.Option[int64]
		}{
			{
				ptr:    nil,
				option: opt.Option[int64]{},
			},
			{
				ptr:    new(int64),
				option: opt.From(int64(0)),
			},
			{
				ptr:    ptr(int64(1)),
				option: opt.From(int64(1)),
			},
			{
				ptr:    ptr(int64(-1)),
				option: opt.From(int64(-1)),
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c.ptr), func(t *testing.T) {
				optData, optErr := json.Marshal(c.option)
				sqlData, sqlErr := json.Marshal(c.ptr)

				assertErrorEq(t, optErr, sqlErr)
				assertBytesEq(t, optData, sqlData)
			})
		}
	})

	t.Run("json.Unmarshaler", func(t *testing.T) {
		cases := []string{
			"",
			"null",
			"nil",
			"{}",
			"\"\"",
			"\"abc\"",
			"\"123\"",
			"0",
			"123",
			"-123",
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				var optInt opt.Option[int64]
				var ptr *int64

				optErr := json.Unmarshal([]byte(c), &optInt)
				ptrErr := json.Unmarshal([]byte(c), &ptr)

				assertErrorEq(t, optErr, ptrErr)
				assertEq(t, optInt.Valid, ptr != nil)
				if ptr != nil {
					assertEq(t, optInt.V, *ptr)
				}
			})
		}
	})
}

func TestOptionFloat64(t *testing.T) {
	t.Run("sql.Scanner", func(t *testing.T) {
		cases := []any{
			float64(0),
			float64(1),
			float64(0.0),
			float64(1.0),
			float64(1.1),
			true,
			false,
			[]byte(nil),
			[]byte{},
			[]byte("hello"),
			"",
			"hello",
			time.Now(),
			time.Time{},
			nil,
		}

		for _, v := range cases {
			t.Run(fmt.Sprint(v), func(t *testing.T) {
				var sqlFloat sql.NullFloat64
				var optFloat opt.Option[float64]

				sqlErr := sqlFloat.Scan(v)
				optErr := optFloat.Scan(v)

				assertErrorEq(t, optErr, sqlErr)
				assertEq(t, optFloat.Valid, sqlFloat.Valid)
				assertEq(t, optFloat.V, sqlFloat.Float64)
			})
		}
	})

	t.Run("driver.Valuer", func(t *testing.T) {
		cases := []struct {
			sql    sql.NullFloat64
			option opt.Option[float64]
		}{
			{
				sql:    sql.NullFloat64{},
				option: opt.Option[float64]{},
			},
			{
				sql:    sql.NullFloat64{Valid: true, Float64: 0},
				option: opt.From(float64(0)),
			},
			{
				sql:    sql.NullFloat64{Valid: true, Float64: 1},
				option: opt.From(float64(1)),
			},
			{
				sql:    sql.NullFloat64{Valid: true, Float64: -1},
				option: opt.From(float64(-1)),
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c.option), func(t *testing.T) {
				sqlVal, sqlErr := c.sql.Value()
				optVal, optErr := c.option.Value()

				assertErrorEq(t, optErr, sqlErr)
				assertEq(t, optVal, sqlVal)
			})
		}
	})

	t.Run("json.Marshaler", func(t *testing.T) {
		cases := []struct {
			ptr    *float64
			option opt.Option[float64]
		}{
			{
				ptr:    nil,
				option: opt.Option[float64]{},
			},
			{
				ptr:    new(float64),
				option: opt.From(float64(0)),
			},
			{
				ptr:    ptr(float64(1)),
				option: opt.From(float64(1)),
			},
			{
				ptr:    ptr(float64(-1)),
				option: opt.From(float64(-1)),
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c.ptr), func(t *testing.T) {
				optData, optErr := json.Marshal(c.option)
				sqlData, sqlErr := json.Marshal(c.ptr)

				assertErrorEq(t, optErr, sqlErr)
				assertBytesEq(t, optData, sqlData)
			})
		}
	})

	t.Run("json.Unmarshaler", func(t *testing.T) {
		cases := []string{
			"",
			"null",
			"nil",
			"{}",
			"\"\"",
			"\"abc\"",
			"\"123\"",
			"0",
			"123",
			"-123",
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				var optFloat opt.Option[float64]
				var ptr *float64

				optErr := json.Unmarshal([]byte(c), &optFloat)
				ptrErr := json.Unmarshal([]byte(c), &ptr)

				assertErrorEq(t, optErr, ptrErr)
				assertEq(t, optFloat.Valid, ptr != nil)
				if ptr != nil {
					assertEq(t, optFloat.V, *ptr)
				}
			})
		}
	})
}

func TestOptionString(t *testing.T) {
	t.Run("sql.Scanner", func(t *testing.T) {
		cases := []any{
			int64(0),
			int64(1),
			float64(0.0),
			float64(1.0),
			float64(1.1),
			true,
			false,
			[]byte(nil),
			[]byte{},
			[]byte("hello"),
			"",
			"hello",
			time.Now(),
			time.Time{},
			nil,
		}

		for _, v := range cases {
			t.Run(fmt.Sprint(v), func(t *testing.T) {
				var sqlStr sql.NullString
				var optStr opt.Option[string]

				sqlErr := sqlStr.Scan(v)
				optErr := optStr.Scan(v)

				assertErrorEq(t, optErr, sqlErr)
				assertEq(t, optStr.Valid, sqlStr.Valid)
				assertEq(t, optStr.V, sqlStr.String)
			})
		}
	})

	t.Run("driver.Valuer", func(t *testing.T) {
		cases := []struct {
			sql    sql.NullString
			option opt.Option[string]
		}{
			{
				sql:    sql.NullString{},
				option: opt.Option[string]{},
			},
			{
				sql:    sql.NullString{Valid: true, String: ""},
				option: opt.From(""),
			},
			{
				sql:    sql.NullString{Valid: true, String: "hello"},
				option: opt.From("hello"),
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c.option), func(t *testing.T) {
				sqlVal, sqlErr := c.sql.Value()
				optVal, optErr := c.option.Value()

				assertErrorEq(t, optErr, sqlErr)
				assertEq(t, optVal, sqlVal)
			})
		}
	})

	t.Run("json.Marshaler", func(t *testing.T) {
		cases := []struct {
			ptr    *string
			option opt.Option[string]
		}{
			{
				ptr:    nil,
				option: opt.Option[string]{},
			},
			{
				ptr:    ptr(""),
				option: opt.From(""),
			},
			{
				ptr:    ptr("hello"),
				option: opt.From("hello"),
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c.ptr), func(t *testing.T) {
				optData, optErr := json.Marshal(c.option)
				sqlData, sqlErr := json.Marshal(c.ptr)

				assertErrorEq(t, optErr, sqlErr)
				assertBytesEq(t, optData, sqlData)
			})
		}
	})

	t.Run("json.Unmarshaler", func(t *testing.T) {
		cases := []string{
			"",
			"null",
			"nil",
			"{}",
			"\"\"",
			"\"abc\"",
			"\"123\"",
			"0",
			"123",
			"-123",
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				var optStr opt.Option[string]
				var ptr *string

				optErr := json.Unmarshal([]byte(c), &optStr)
				ptrErr := json.Unmarshal([]byte(c), &ptr)

				assertErrorEq(t, optErr, ptrErr)
				assertEq(t, optStr.Valid, ptr != nil)
				if ptr != nil {
					assertEq(t, optStr.V, *ptr)
				}
			})
		}
	})
}

func TestOptionBool(t *testing.T) {
	t.Run("sql.Scanner", func(t *testing.T) {
		cases := []any{
			int64(0),
			int64(1),
			float64(0.0),
			float64(1.0),
			float64(1.1),
			true,
			false,
			[]byte(nil),
			[]byte{},
			[]byte("hello"),
			"",
			"hello",
			time.Now(),
			time.Time{},
			nil,
		}

		for _, v := range cases {
			t.Run(fmt.Sprint(v), func(t *testing.T) {
				var sqlStr sql.NullBool
				var optStr opt.Option[bool]

				sqlErr := sqlStr.Scan(v)
				optErr := optStr.Scan(v)

				assertErrorEq(t, optErr, sqlErr)
				assertEq(t, optStr.Valid, sqlStr.Valid)
				assertEq(t, optStr.V, sqlStr.Bool)
			})
		}
	})

	t.Run("driver.Valuer", func(t *testing.T) {
		cases := []struct {
			sql    sql.NullBool
			option opt.Option[bool]
		}{
			{
				sql:    sql.NullBool{},
				option: opt.Option[bool]{},
			},
			{
				sql:    sql.NullBool{Valid: true, Bool: true},
				option: opt.From(true),
			},
			{
				sql:    sql.NullBool{Valid: true, Bool: false},
				option: opt.From(false),
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c.option), func(t *testing.T) {
				sqlVal, sqlErr := c.sql.Value()
				optVal, optErr := c.option.Value()

				assertErrorEq(t, optErr, sqlErr)
				assertEq(t, optVal, sqlVal)
			})
		}
	})

	t.Run("json.Marshaler", func(t *testing.T) {
		cases := []struct {
			ptr    *bool
			option opt.Option[bool]
		}{
			{
				ptr:    nil,
				option: opt.Option[bool]{},
			},
			{
				ptr:    ptr(true),
				option: opt.From(true),
			},
			{
				ptr:    ptr(false),
				option: opt.From(false),
			},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c.ptr), func(t *testing.T) {
				optData, optErr := json.Marshal(c.option)
				sqlData, sqlErr := json.Marshal(c.ptr)

				assertErrorEq(t, optErr, sqlErr)
				assertBytesEq(t, optData, sqlData)
			})
		}
	})

	t.Run("json.Unmarshaler", func(t *testing.T) {
		cases := []string{
			"",
			"null",
			"nil",
			"{}",
			"\"\"",
			"\"abc\"",
			"\"123\"",
			"0",
			"123",
			"-123",
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				var optStr opt.Option[bool]
				var ptr *bool

				optErr := json.Unmarshal([]byte(c), &optStr)
				ptrErr := json.Unmarshal([]byte(c), &ptr)

				assertErrorEq(t, optErr, ptrErr)
				assertEq(t, optStr.Valid, ptr != nil)
				if ptr != nil {
					assertEq(t, optStr.V, *ptr)
				}
			})
		}
	})
}

type TestStruct1 struct {
	V string
}

type TestStruct2 struct {
	V string
}

var (
	_ json.Marshaler   = TestStruct2{}
	_ json.Unmarshaler = &TestStruct2{}
	_ driver.Valuer    = TestStruct2{}
	_ sql.Scanner      = &TestStruct2{}
)

func (t TestStruct2) MarshalJSON() ([]byte, error) {
	return []byte(t.V), nil
}

func (t *TestStruct2) UnmarshalJSON(data []byte) error {
	t.V = string(data)
	return nil
}

func (t TestStruct2) Value() (driver.Value, error) {
	return t.V, nil
}

func (t *TestStruct2) Scan(data any) error {
	switch v := data.(type) {
	case string:
		t.V = v
	case []byte:
		t.V = string(v)
	}

	return errors.New("scan failed")
}

func TestOptionStruct1(t *testing.T) {
	t.Run("driver.Valuer", func(t *testing.T) {
		cases := []*TestStruct1{
			nil,
			{},
			{"hello"},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				ptrVal, ptrErr := driver.DefaultParameterConverter.ConvertValue(c)
				optVal, optErr := driver.DefaultParameterConverter.ConvertValue(opt.FromPtr(c))

				assertErrorEq(t, optErr, ptrErr)
				assertEq(t, optVal, ptrVal)
			})
		}
	})

	t.Run("json.Marshaler", func(t *testing.T) {
		cases := []*TestStruct1{
			nil,
			{},
			{"hello"},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				optData, optErr := json.Marshal(c)
				sqlData, sqlErr := json.Marshal(opt.FromPtr(c))

				assertErrorEq(t, optErr, sqlErr)
				assertBytesEq(t, optData, sqlData)
			})
		}
	})

	t.Run("json.Unmarshaler", func(t *testing.T) {
		cases := []string{
			"",
			"null",
			"nil",
			"{}",
			"\"\"",
			"\"abc\"",
			"\"123\"",
			"0",
			"123",
			"-123",
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				var optStruct opt.Option[TestStruct1]
				var ptr *TestStruct1

				optErr := json.Unmarshal([]byte(c), &optStruct)
				ptrErr := json.Unmarshal([]byte(c), &ptr)

				assertErrorEq(t, optErr, ptrErr)
				assertEq(t, optStruct.Valid, ptr != nil)
				if ptr != nil {
					assertEq(t, optStruct.V, *ptr)
				}
			})
		}
	})
}

func TestOptionStruct2(t *testing.T) {
	t.Run("driver.Valuer", func(t *testing.T) {
		cases := []*TestStruct2{
			nil,
			{},
			{"hello"},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				ptrVal, ptrErr := driver.DefaultParameterConverter.ConvertValue(c)
				optVal, optErr := driver.DefaultParameterConverter.ConvertValue(opt.FromPtr(c))

				assertErrorEq(t, optErr, ptrErr)
				assertEq(t, optVal, ptrVal)
			})
		}
	})

	t.Run("json.Marshaler", func(t *testing.T) {
		cases := []*TestStruct2{
			nil,
			{},
			{"hello"},
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				optData, optErr := json.Marshal(c)
				sqlData, sqlErr := json.Marshal(opt.FromPtr(c))

				assertEq(t, optErr == nil, sqlErr == nil)
				assertBytesEq(t, optData, sqlData)
			})
		}
	})

	t.Run("json.Unmarshaler", func(t *testing.T) {
		cases := []string{
			"",
			"null",
			"nil",
			"{}",
			"\"\"",
			"\"abc\"",
			"\"123\"",
			"0",
			"123",
			"-123",
		}

		for _, c := range cases {
			t.Run(fmt.Sprint(c), func(t *testing.T) {
				var optStruct opt.Option[TestStruct2]
				var ptr *TestStruct2

				optErr := json.Unmarshal([]byte(c), &optStruct)
				ptrErr := json.Unmarshal([]byte(c), &ptr)

				assertErrorEq(t, optErr, ptrErr)
				assertEq(t, optStruct.Valid, ptr != nil)
				if ptr != nil {
					assertEq(t, optStruct.V, *ptr)
				}
			})
		}
	})
}

func TestOptionScanAssign(t *testing.T) {
	t.Run("string to []byte", func(t *testing.T) {
		o := opt.New[[]byte]()
		if err := o.Scan("hello"); err != nil {
			t.Error(err)
		}
		assertEq(t, string(o.V), "hello")
	})

	t.Run("[]byte to []byte", func(t *testing.T) {
		o := opt.New[[]byte]()
		if err := o.Scan([]byte("hello")); err != nil {
			t.Error(err)
		}
		assertEq(t, string(o.V), "hello")
	})

	t.Run("[]byte to any", func(t *testing.T) {
		o := opt.New[any]()
		if err := o.Scan([]byte("hello")); err != nil {
			t.Error(err)
		}
		assertBytesEq(t, o.V.([]byte), []byte("hello"))
	})

	t.Run("Time to Time", func(t *testing.T) {
		o := opt.New[time.Time]()
		now := time.Now()
		if err := o.Scan(now); err != nil {
			t.Error(err)
		}
		assertEq(t, o.V, now)
	})

	t.Run("Time to []byte", func(t *testing.T) {
		o := opt.New[[]byte]()
		now := time.Now()
		if err := o.Scan(now); err != nil {
			t.Error(err)
		}
		assertBytesEq(t, o.V, []byte(now.Format(time.RFC3339Nano)))
	})

	t.Run("int to any", func(t *testing.T) {
		o := opt.New[any]()
		if err := o.Scan(1); err != nil {
			t.Error(err)
		}
		assertEq(t, o.V, 1)
	})

	t.Run("int to sql.Scanner", func(t *testing.T) {
		o := opt.New[sql.NullInt64]()
		if err := o.Scan(1); err != nil {
			t.Error(err)
		}
		assertEq(t, o.V, sql.NullInt64{Valid: true, Int64: 1})
	})

	t.Run("bytes assignable", func(t *testing.T) {
		o := opt.New[json.RawMessage]()
		if err := o.Scan([]byte("hello")); err != nil {
			t.Error(err)
		}
		assertBytesEq(t, o.V, []byte("hello"))
	})
}

func ptr[T any](v T) *T { return &v }

func assertEq[T comparable](t *testing.T, actual, expected T) {
	t.Helper()

	if actual != expected {
		t.Errorf("expected %#v, got %#v", expected, actual)
	}
}

func assertErrorEq(t *testing.T, actual, expected error) {
	t.Helper()

	if (expected == nil) != (actual == nil) || (expected != nil && actual != nil && expected.Error() != actual.Error()) {
		t.Errorf("expected %v, got %v", expected, actual)
	}
}

func assertBytesEq(t *testing.T, actual, expected []byte) {
	t.Helper()

	if (expected == nil) != (actual == nil) || !bytes.Equal(actual, expected) {
		t.Errorf("expected %s, got %s", expected, actual)
	}
}
