/*

fml is library for parsing MARC 21 formatted data.

*/
package fml

import (
	"bufio"
	"bytes"
	"errors"
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
	Data   []byte
	Fields []interface{}
	Leader Leader
}

// Leader contains a subset of the bytes in the record leader. Omitted are
// bytes specifying the length of parts of the record and bytes which do
// not vary from record to record.
type Leader struct {
	Status        byte // 05 byte position
	Type          byte // 06
	BibLevel      byte // 07
	Control       byte // 08
	EncodingLevel byte // 17
	Form          byte // 18
	Multipart     byte // 19
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

// ControlNum returns the record's control number.
func (r Record) ControlNum() string {
	cf := r.ControlField("001")[0]
	return strings.TrimSpace(cf.Value)
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

func (d DataField) matches(tag string, ind1 string, ind2 string) bool {
	t := d.Tag == tag
	i1 := ind1 == "*" || d.Indicator1 == ind1
	i2 := ind2 == "*" || d.Indicator2 == ind2
	return t && i1 && i2
}

// Filter takes one or more tag queries and returns a slice of strings
// matching the selected subfield values. A tag query consists of the
// three digit MARC tag optionally followed by one or more subfield codes,
// for example: "245ac", "650x" or "100". Filtering for indicators can be
// done by including the two desired indicators between pipes after the tag.
// An * character can be used for any inidicator, for example: "245|*1|ac"
// or 650|01|x.
func (r Record) Filter(query ...string) [][]string {
	var res [][]string
	for _, q := range query {
		tag := q[:3]
		for _, field := range r.Fields {
			var values []string
			switch f := field.(type) {
			case ControlField:
				if f.Tag == tag {
					values = append(values, f.Value)
					res = append(res, values)
				}
			case DataField:
				ind := strings.Index(q, "|")
				var subs string
				ind1, ind2 := "*", "*"
				if ind > -1 {
					ind1, ind2 = string(q[ind+1]), string(q[ind+2])
					subs = q[ind+4:]
				} else {
					subs = q[3:]
				}
				if f.matches(tag, ind1, ind2) {
					if len(subs) != 0 {
						for _, sf := range f.SubField(strings.Split(subs, "")...) {
							values = append(values, sf.Value)
						}
					} else {
						for _, sf := range f.SubFields {
							values = append(values, sf.Value)
						}
					}
					if len(values) > 0 {
						res = append(res, values)
					}
				}
			}
		}
	}
	return res
}

// Next advances the MarcIterator to the next record, which will be
// available through the Value method. It returns false when the
// MarcIterator has reached the end of the file or has encountered an error.
// Any error will be accessible from the Err method.
func (m *MarcIterator) Next() bool {
	return m.scanner.Scan()
}

// Value returns the current Record or the MarcIterator.
func (m *MarcIterator) Value() (Record, error) {
	return m.scanIntoRecord(m.scanner.Bytes())
}

// Err will return the first error encountered by the MarcIterator.
func (m *MarcIterator) Err() error {
	return m.scanner.Err()
}

func (m *MarcIterator) scanIntoRecord(bytes []byte) (Record, error) {
	rec := Record{}
	rec.Data = append([]byte(nil), bytes...)
	rec.Leader = Leader{
		Status:        bytes[5],
		Type:          bytes[6],
		BibLevel:      bytes[7],
		Control:       bytes[8],
		EncodingLevel: bytes[17],
		Form:          bytes[18],
		Multipart:     bytes[19],
	}

	start, err := strconv.Atoi(string(bytes[12:17]))
	if err != nil {
		return rec, errors.New("Could not determine record start")
	}
	data := bytes[start:]
	dirs := bytes[24 : start-1]

	for len(dirs) >= 12 {
		tag := string(dirs[:3])
		length, err := strconv.Atoi(string(dirs[3:7]))
		if err != nil {
			return rec, errors.New("Could not determine length of field")
		}
		begin, err := strconv.Atoi(string(dirs[7:12]))
		if err != nil {
			return rec, errors.New("Could not determine field start")
		}
		if len(data) <= begin+length-1 {
			return rec, errors.New("Reported field length incorrect")
		}
		fdata := data[begin : begin+length-1] // length includes field terminator
		if strings.HasPrefix(tag, "00") {
			rec.Fields = append(rec.Fields, ControlField{tag, string(fdata)})
		} else {
			df, err := makeDataField(tag, fdata)
			if err != nil {
				return rec, err
			}
			rec.Fields = append(rec.Fields, df)
		}
		dirs = dirs[12:]
	}
	return rec, nil
}

// NewMarcIterator creates and returns a new instance of a MarcIterator.
// This function should be used to create a MarcIterator rather than
// instantiating one yourself.
func NewMarcIterator(r io.Reader) *MarcIterator {
	scanner := bufio.NewScanner(r)
	scanner.Split(splitFunc)
	return &MarcIterator{scanner}
}

func makeDataField(tag string, data []byte) (DataField, error) {
	d := DataField{}
	d.Tag = tag
	if len(data) > 2 {
		d.Indicator1 = string(data[0])
		d.Indicator2 = string(data[1])
	} else {
		return d, errors.New("Invalid Indicators detected")
	}
	for _, sf := range bytes.Split(data[3:], []byte{st}) {
		if len(sf) > 0 {
			d.SubFields = append(d.SubFields, SubField{string(sf[0]), string(sf[1:])})
		} else {
			return d, errors.New("Extraneous field terminator")
		}
	}
	return d, nil
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
