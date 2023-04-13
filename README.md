# Option[T]

Package `opt` contains type `Option[T]` which a generic wrapper for optional values compatible with `encoding/json` and `database/sql`.

If your app is regularly moving optional values through a JSON interface and in/out of a database, this type can make your code more readable, less verbose, and safer.

## Reasoning

- Explicit Optionality
- Avoid pointers for optional values
    - Improve readability
    - Prevent accidental nil pointer dereferences
    - Avoid implicit pass-by-reference for optional values
- Avoid tedious and verbose conversions between `sql.NullString` and `*string` etc.

## Usage

### Example

```go
type MyType struct {
	Value opt.Option[int] `json:"value"`
}

func OptionJSONExample() {
	jsonMessage := []byte(`{"value": null}`)

	var myType MyType
	json.Unmarshal(jsonMessage, &myType)

	fmt.Println("json.Unmarshal result:", myType)

	data, _ := json.Marshal(myType)

	fmt.Println("json.Marshal result:", string(data))
}

func OptionSQLExample() {
	db, _ := sql.Open("", "")

	db.Exec(`CREATE TABLE vals (value INTEGER);`)

	myType := MyType{
		Value: opt.New[int](),
	}

	db.Exec("INSERT INTO vals (value) VALUES ($1);", myType.Value)

	db.QueryRow("SELECT value FROM vals;").Scan(&myType.Value)

	fmt.Println("query result:", myType)
}

```

#### Output

```
json.Unmarshal result: {null}
json.Marshal result: {"value":null}
query result: {null}
```

### Details

- A null `Option[T]` will be printed as `null` by `fmt.Sprint`
- A non-null `Option[T]` will be printed as if it were the value.
- The zero value for an Option is `null`.
- The struct fields `Valid` and `V` are public for reading. Writing to them is safe but not encouraged.
    - Instead use `New` and `From` to set new values.

## Compatibility

Currently compatibility is only provided for `encoding/json` and `database/sql`.
This could be extended in the future, for example to `encoding/xml`.

### JSON

For all intents and purposes, JSON marshalling and unmarshalling works the same as using a pointer.

If you want to convert from/to pointers, use `FromPtr` and `(opt.Option).Ptr`, respectively.

Note that `T` must be a type that is itself compatible with `encoding/json`.
You can implement this on custom types by implementing `json.Marshaler` and `json.Unmarshaler`

### SQL

For all intents and purposes, sql scanning and writing works the same as `sql.NullInt64`, `sql.NullString`, etc.

Note that `T` must be a type that is itself compatible with `database/sql`.
You can implement this on custom types by implementing `sql.Scanner` and `driver.Valuer`