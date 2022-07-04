# opt

The opt package contains the Option type which is a generic wrapper to make other types nullable. It is especially useful for encoding, and replaces the pointers commonly used to distinguish between null and zero values.

## Compatibilty

* `database/sql`
* `encoding/json`
* `encoding/xml`

## Usage

```go
func instantiation() {
    // instantiate a null value
    var null1 opt.Option[int]
    null2 := opt.Option[int]{}
    null2 := opt.FromPtr[int](nil)

    // instantiate a non-null value
    notNull1 := opt.From(0)
    notNull2 := opt.FromPtr(new(0))
}

func usage(v opt.Option[int]) {
    v.Null()       // bool
    v.NullOrZero() // bool
    v.Get()        // int
}

// compatible with encoding/json & encoding/xml
type myType struct{
    Field opt.Option[int] `json:"nullable_int" xml:"nullable_int"`
}

// compatible with database/sql (and database/sql/driver)
func scanSQL(rows *sql.Rows) string {
    var dst opt.Option[string]
    _ = rows.Scan(&dst)
    return dst.Get()
}
```
