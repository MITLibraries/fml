# fml (Fast MARC Library)

[![GoDoc](https://godoc.org/github.com/MITLibraries/fml?status.svg)](https://godoc.org/github.com/MITLibraries/fml)

fml is a Go library for parsing MARC 21 formatted data. The library interface should still be considered unstable and may change in backwards incompatible ways.

There is also an `fml` command line utility that can be used to pull a single MARC record from a file by control number. The command can be installed with:

```
$ go get github.com/mitlibraries/fml/cmd/fml
```

## How do I use this?

```go
import "github.com/mitlibraries/fml"
```

Start by creating a new `MarcIterator`:

```go
m := fml.NewMarcIterator(<io.Reader>)
```

A `MarcIterator` can be iterated over by using the `Next()` and `Value()` methods. This is mostly just a thin wrapper around a `bufio.Scanner`. `Next()` returns false when there is no more data to process or an unrecoverable error has occured. In this case, iteration will stop and the error will be available from the `Err()` method:

```go
for m.Next() {
  record, err := m.Value()
  // do something with record
}
err := m.Err()
if err != nil {
  // do something with error
}
```

A `Record` contains a leader struct and a slice of all the control and data fields. If you want to iterate over this slice you will need to use a type switch to determine the type of field. There are a few convenience methods that are probably better for accessing specific fields, though.

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

There is also `Filter` method inspired by [traject](https://github.com/traject/traject). `Filter` takes one or more query strings consisting of a three digit MARC tag optionally followed by an indicator filter and/or one or more subfield codes. If no subfields are specified, all subfields for the matching tag are returned. The indicator filter consists of two indicator codes between pipes. A `*` character can be used to match any indicator code. Here are a few examples of valid query strings:

```
200
500|02|
650x
245|*1|ac
```

`Filter` returns a slice of string slices. Each slice member represents an instance of a matching tag and a slice of all the matching subfields, or data values in the case of control fields. For example, take a MARC record with the following structure:

```
245 $a Tomb of Annihilation
650 $a Dungeons and Dragons (Game) $v Handbooks, manuals, etc.
650 $a Dungeons and Dragons (Game) $v Rules.
```

The following code:

```go
for _, t := range record.Filter("245", "650v") {
  for _, v := range t {
    fmt.Println(v)
  }
}
```

outputs:

```
Tomb of Annihilation
Handbooks, manuals, etc.
Rules.
```
