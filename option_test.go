package opt_test

import (
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"testing"

	"git.ultraware.nl/martin/opt"
	"git.ultraware.nl/martin/opt/internal/test/assert"
)

func TestFrom(t *testing.T) {
	t.Run(`non-zero value`, func(t *testing.T) {
		o := opt.From(1)
		assert.Equal(t, 1, o.Get())
		assert.Equal(t, false, o.Null())
		assert.Equal(t, false, o.NullOrZero())
	})

	t.Run(`zero value`, func(t *testing.T) {
		o := opt.From(0)
		assert.Equal(t, 0, o.Get())
		assert.Equal(t, false, o.Null())
		assert.Equal(t, true, o.NullOrZero())
	})
}

func TestFromPtr(t *testing.T) {
	t.Run(`non-zero value`, func(t *testing.T) {
		v := 1
		o := opt.FromPtr(&v)
		assert.Equal(t, 1, o.Get())
		assert.Equal(t, false, o.Null())
		assert.Equal(t, false, o.NullOrZero())
	})

	t.Run(`zero value`, func(t *testing.T) {
		v := 0
		o := opt.FromPtr(&v)
		assert.Equal(t, o.Get(), 0)
		assert.Equal(t, o.Null(), false)
		assert.Equal(t, o.NullOrZero(), true)
	})

	t.Run(`null`, func(t *testing.T) {
		o := opt.FromPtr[int](nil)
		assert.Equal(t, o.Get(), 0)
		assert.Equal(t, o.Null(), true)
		assert.Equal(t, o.NullOrZero(), true)
	})
}

func TestNullOrZero(t *testing.T) {
	t.Run(`comparable`, func(t *testing.T) {
		o := opt.From(0)
		assert.Equal(t, o.NullOrZero(), true)
		o = opt.From(1)
		assert.Equal(t, o.NullOrZero(), false)
		o = opt.FromPtr[int](nil)
		assert.Equal(t, o.NullOrZero(), true)
	})

	t.Run(`interface`, func(t *testing.T) {
		o := opt.From(reflect.TypeOf(1))
		assert.Equal(t, o.NullOrZero(), false)
		var a reflect.Type
		o = opt.From(a)
		assert.Equal(t, o.NullOrZero(), true)
		o = opt.FromPtr[reflect.Type](nil)
		assert.Equal(t, o.NullOrZero(), true)
	})

	t.Run(`non-comparable`, func(t *testing.T) {
		o := opt.From(make(map[int]int))
		assert.Equal(t, o.NullOrZero(), false)
		var a map[int]int
		o = opt.From(a)
		assert.Equal(t, o.NullOrZero(), true)
		o = opt.FromPtr[map[int]int](nil)
		assert.Equal(t, o.NullOrZero(), true)
	})
}

func TestString(t *testing.T) {
	t.Run(`null string`, func(t *testing.T) {
		o := opt.FromPtr[string](nil)
		assert.Equal(t, o.String(), `null`)
	})

	t.Run(`empty string`, func(t *testing.T) {
		o := opt.From(``)
		assert.Equal(t, o.String(), ``)
	})

	t.Run(`filled string`, func(t *testing.T) {
		o := opt.From(`bla`)
		assert.Equal(t, o.String(), `bla`)
	})
}

func TestGoString(t *testing.T) {
	t.Run(`null string`, func(t *testing.T) {
		o := opt.FromPtr[string](nil)
		assert.Equal(t, o.GoString(), `opt.opt[string]{}`)
	})

	t.Run(`empty string`, func(t *testing.T) {
		o := opt.From(``)
		assert.Equal(t, o.GoString(), `opt.From[string]("")`)
	})

	t.Run(`filled string`, func(t *testing.T) {
		o := opt.From(`bla`)
		assert.Equal(t, o.GoString(), `opt.From[string]("bla")`)
	})

	type myType struct{}

	t.Run(`path prefix filled correctly for non primitives`, func(t *testing.T) {
		o := opt.From(myType{})
		assert.Equal(t, o.GoString(), `opt.From[opt_test.myType](opt_test.myType{})`)
	})
}

func TestSQL(t *testing.T) {
	o := opt.Option[int]{}
	actual, err := o.Value()
	assert.NoError(t, err)
	expected, err := driver.DefaultParameterConverter.ConvertValue(nil)
	assert.NoError(t, err)
	assert.AnyEqual(t, actual, expected)

	o = opt.From(3)
	actual, err = o.Value()
	assert.NoError(t, err)
	expected, err = driver.DefaultParameterConverter.ConvertValue(3)
	assert.NoError(t, err)
	assert.AnyEqual(t, actual, expected)

	o = opt.Option[int]{}
	assert.NoError(t, o.Scan(nil))
	assert.Equal(t, o, opt.Option[int]{})

	o = opt.Option[int]{}
	assert.NoError(t, o.Scan(3))
	assert.Equal(t, o, opt.From(3))
}

func TestMarshalJSON(t *testing.T) {
	t.Run(`marshal null`, func(t *testing.T) {
		o := opt.FromPtr[string](nil)
		data, err := json.Marshal(o)
		assert.NoError(t, err)
		assert.Equal(t, string(data), `null`)
	})

	t.Run(`marshal non-null`, func(t *testing.T) {
		o := opt.From(``)
		data, err := json.Marshal(o)
		assert.NoError(t, err)
		assert.Equal(t, string(data), `""`)
	})

	type bla struct {
		A int `json:"a"`
	}

	t.Run(`marshal struct`, func(t *testing.T) {
		o := opt.From(bla{1})
		data, err := json.Marshal(o)
		assert.NoError(t, err)
		assert.Equal(t, string(data), `{"a":1}`)
	})
}

func TestUnmarshalJSON(t *testing.T) {
	type jsonType struct {
		O opt.Option[string] `json:"a"`
	}

	t.Run(`unmarshal null`, func(t *testing.T) {
		var o jsonType
		err := json.Unmarshal([]byte(`{"a": null}`), &o)
		assert.NoError(t, err)
		assert.Equal(t, o.O.Null(), true)
		assert.Equal(t, o.O.Get(), ``)
	})

	t.Run(`unmarshal nothing`, func(t *testing.T) {
		var o jsonType
		err := json.Unmarshal([]byte(`{}`), &o)
		assert.NoError(t, err)
		assert.Equal(t, o.O.Null(), true)
		assert.Equal(t, o.O.Get(), ``)
	})

	t.Run(`unmarshal value`, func(t *testing.T) {
		var o jsonType
		err := json.Unmarshal([]byte(`{"a": ""}`), &o)
		assert.NoError(t, err)
		assert.Equal(t, o.O.Null(), false)
		assert.Equal(t, o.O.Get(), ``)
	})
}

type xmlType struct {
	XMLName xml.Name        `xml:"a"`
	O       opt.Option[int] `xml:"b"`
}

func TestMarshalXML(t *testing.T) {
	t.Run(`marshal null`, func(t *testing.T) {
		o := xmlType{
			O: opt.FromPtr[int](nil),
		}
		data, err := xml.Marshal(o)
		assert.NoError(t, err)
		assert.Equal(t, string(data), `<a></a>`)
	})

	t.Run(`marshal non-null`, func(t *testing.T) {
		o := xmlType{
			O: opt.From(1),
		}
		data, err := xml.Marshal(o)
		assert.NoError(t, err)
		assert.Equal(t, string(data), `<a><b>1</b></a>`)
	})
}

func TestUnmarshalXML(t *testing.T) {
	t.Run(`unmarshal null`, func(t *testing.T) {
		var o xmlType
		err := xml.Unmarshal([]byte(`<a></a>`), &o)
		assert.NoError(t, err)
		assert.Equal(t, o.O.Null(), true)
		assert.Equal(t, o.O.Get(), 0)
	})

	t.Run(`unmarshal value`, func(t *testing.T) {
		var o xmlType
		err := xml.Unmarshal([]byte(`<a><b>1</b></a>`), &o)
		assert.NoError(t, err)
		assert.Equal(t, o.O.Null(), false)
		assert.Equal(t, o.O.Get(), 1)
	})
}
