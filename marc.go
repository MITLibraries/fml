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
	Leader Leader
	fields map[string][][]byte
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
		for _, f := range r.fields[t] {
			df := DataField{string(f[0]), string(f[1]), t, subfields(f[3:])}
			fields = append(fields, df)
		}
	}
	return fields
}

// ControlField method takes an arbitrary number of tag strings and returns
// a slice of matching ControlFields.
func (r Record) ControlField(tag ...string) []ControlField {
	fields := make([]ControlField, 0, len(tag))
	for _, t := range tag {
		for _, f := range r.fields[t] {
			fields = append(fields, ControlField{t, string(f)})
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

func matches(data []byte, ind1 byte, ind2 byte) bool {
	i1 := ind1 == '*' || data[0] == ind1
	i2 := ind2 == '*' || data[1] == ind2
	return i1 && i2
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
		tag, ind1, ind2, subs := splitQuery(q)
		for _, field := range r.fields[tag] {
			var values []string
			if strings.HasPrefix(tag, "00") {
				values = append(values, string(field))
			} else {
				if matches(field, ind1, ind2) {
					for _, sf := range subfields(field[3:], strings.Split(subs, "")...) {
						values = append(values, sf.Value)
					}
				}
			}
			if len(values) > 0 {
				res = append(res, values)
			}
		}
	}
	return res
}

func subfields(data []byte, codes ...string) []SubField {
	var res []SubField
	for _, sf := range bytes.Split(data, []byte{st}) {
		if len(sf) == 0 {
			continue
		}
		if len(codes) > 0 {
			for _, code := range codes {
				if string(sf[0]) == code {
					res = append(res, SubField{string(sf[0]), string(sf[1:])})
					break
				}
			}
		} else {
			res = append(res, SubField{string(sf[0]), string(sf[1:])})
		}
	}
	return res
}

func splitQuery(query string) (string, byte, byte, string) {
	var subs string
	ind1, ind2 := byte('*'), byte('*')
	tag := query[:3]
	ind := strings.Index(query, "|")
	if ind > -1 {
		ind1, ind2 = query[ind+1], query[ind+2]
		subs = query[ind+4:]
	} else {
		subs = query[3:]
	}
	return tag, ind1, ind2, subs
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
	rec.fields = make(map[string][][]byte)
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

	for len(dirs) > 0 {
		tag := string(dirs[:3])
		length, err := strconv.Atoi(string(dirs[3:7]))
		if err != nil {
			return rec, errors.New("Could not determine length of field")
		}
		begin, err := strconv.Atoi(string(dirs[7:12]))
		if err != nil {
			return rec, errors.New("Could not determine field start")
		}
		fdata := data[begin : begin+length-1] // length includes field terminator
		fcopy := make([]byte, len(fdata))
		copy(fcopy, fdata)
		rec.fields[tag] = append(rec.fields[tag], fcopy)
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
