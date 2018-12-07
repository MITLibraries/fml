# fml (Fast MARC Library)

fml is a Go library for parsing MARC 21 formatted data.

## How do I use this?

Start by creating a new `MarcIterator`:

```go
m := fml.NewMarcIterator(<io.Reader>)
```

A `MarcIterator` can be iterated over by using the `Next()` and `Value()` methods. This is mostly just a thin wrapper around a `bufio.Scanner`. If an error occured, iteration will stop and the error will be available from the `Err()` method.

```go
for m.Next() {
  record := m.Value()
  // do something with record
}
err := m.Err()
if err != nil {
  // do something with error
}
```

A `Record` contains a slice of all the control and data fields. If you want to iterate over this slice you will want to use a type switch to determine the type of field. There are a few convenience methods that are probably better for accessing specific fields, though.

The `ControlField` method returns a slice of control fields. A control field contains a tag and a value:

```go
for _, cf := range record.ControlField("001", "003") {
  fmt.Printf("%s: %s\n", cf.Tag, cf.Value)
}
```

The `DataField` method returns a slice of data fields. A data field has two indicators, a tag and a slice of subfields. The subfields themselves have a code and a value. There's also a `SubField` method that returns a slice of specified subfields:

```go
for _, df := range record.DataField("260") {
  fmt.Printf("%s: %s %s\n", df.Tag, df.Indicator1, df.Indicator2)
  for _, sf := range df.SubField("a", "b") {
    fmt.Printf("\t%s: %s\n", sf.Code, sf.Value)
  }
}
```
