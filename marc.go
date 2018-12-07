package fml

import (
	"bufio"
	"bytes"
	"io"
	"strconv"
	"strings"
)

const (
	rt = 0x1d // End of record
	st = 0x1f // End of subfield
)

// Record is a struct representing a MARC record. It has a Fields slice
// which contains both ControlFields and DataFields.
type Record struct {
	Fields []interface{}
}

// ControlField just contains a Tag and a Value.
type ControlField struct {
	Tag   string
	Value string
}

// DataField contains two Indicators, a Tag, and a slice of SubFields. If
// you want a specific subfield or subfields you should use the SubField
// method.
type DataField struct {
	Indicator1 string
	Indicator2 string
	Tag        string
	SubFields  []SubField
}

// SubField contains a Code and a Value.
type SubField struct {
	Code  string
	Value string
}

// MarcIterator will iterate over a set of MARC records using the Next()
// and Value() methods. Use the NewMarcIterator function to create a
// MarcIterator.
type MarcIterator struct {
	scanner *bufio.Scanner
}

// DataField method takes an arbitrary number of tag strings and returns
// a slice of matching DataFields. Note that one tag may return multiple
// DataFields as they can be repeated.
func (r Record) DataField(tag ...string) []DataField {
	fields := make([]DataField, 0, len(tag))
	for _, t := range tag {
		for _, f := range r.Fields {
			field, ok := f.(DataField)
			if ok && field.Tag == t {
				fields = append(fields, field)
			}
		}
	}
	return fields
}

// ControlField method takes an arbitrary number of tag strings and returns
// a slice of matching ControlFields.
func (r Record) ControlField(tag ...string) []ControlField {
	fields := make([]ControlField, 0, len(tag))
	for _, t := range tag {
		for _, f := range r.Fields {
			field, ok := f.(ControlField)
			if ok && field.Tag == t {
				fields = append(fields, field)
			}
		}
	}
	return fields
}

// SubField takes an arbitrary number of subfield code strings and returns
// a slice of SubFields.
func (d DataField) SubField(subfield ...string) []SubField {
	fields := make([]SubField, 0, len(subfield))
	for _, s := range subfield {
		for _, f := range d.SubFields {
			if f.Code == s {
				fields = append(fields, f)
			}
		}
	}
	return fields
}

// Next advances the MarcIterator to the next record, which will be
// available through the Value method. It returns false when the
// MarcIterator has reached the end of the file or has encountered an error.
// Any error will be accessible from the Err method.
func (m *MarcIterator) Next() bool {
	return m.scanner.Scan()
}

// Value returns the current Record or the MarcIterator.
func (m *MarcIterator) Value() Record {
	return m.scanIntoRecord(m.scanner.Bytes())
}

// Err will return the first error encountered by the MarcIterator.
func (m *MarcIterator) Err() error {
	return m.scanner.Err()
}

func (m *MarcIterator) scanIntoRecord(bytes []byte) Record {
	rec := Record{}

	start, err := strconv.Atoi(string(bytes[12:17]))
	if err != nil {
		panic(err)
	}
	data := bytes[start:]
	dirs := bytes[24 : start-1]

	for len(dirs) > 0 {
		tag := string(dirs[:3])
		dLength, _ := strconv.Atoi(string(dirs[3:7]))
		dStart, _ := strconv.Atoi(string(dirs[7:12]))
		//Length includes the field terminator
		addField(&rec, tag, data[dStart:dStart+dLength-1])
		dirs = dirs[12:]
	}
	return rec
}

// NewMarcIterator creates and returns a new instance of a MarcIterator.
// This function should be used to create a MarcIterator rather than
// instantiating one yourself.
func NewMarcIterator(r io.Reader) *MarcIterator {
	scanner := bufio.NewScanner(r)
	scanner.Split(splitFunc)
	return &MarcIterator{scanner}
}

func addField(r *Record, tag string, data []byte) {
	if strings.HasPrefix(tag, "00") {
		r.Fields = append(r.Fields, ControlField{tag, string(data)})
	} else {
		r.Fields = append(r.Fields, makeDataField(tag, data))
	}
}

func makeDataField(tag string, data []byte) DataField {
	d := DataField{}
	d.Tag = tag
	d.Indicator1 = string(data[0])
	d.Indicator2 = string(data[1])
	for _, sf := range bytes.Split(data[3:], []byte{st}) {
		d.SubFields = append(d.SubFields, SubField{string(sf[0]), string(sf[1:])})
	}
	return d
}

func splitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if atEOF {
		return len(data), data, nil
	}

	if i := bytes.IndexByte(data, rt); i >= 0 {
		return i + 1, data[0:i], nil
	}
	return
}
