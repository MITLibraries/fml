package fml

import (
	"os"
	"strings"
	"testing"
)

func TestRecord(t *testing.T) {
	f, err := os.Open("fixtures/record1.mrc")
	if err != nil {
		t.Error(err)
	}
	iter := NewMarcIterator(f)
	_ = iter.Next()
	r := iter.Value()
	t.Run("ControlNum", func(t *testing.T) {
		if r.ControlNum() != "92005291" {
			t.Error("Expected 92005291, got", r.ControlNum())
		}
	})
	t.Run("Leader", func(t *testing.T) {
		if r.Leader.Type != 'a' {
			t.Error("Expected a, got", r.Leader.Type)
		}
	})
	t.Run("Specific control field", func(t *testing.T) {
		cf := r.ControlField("001")[0]
		if cf.Tag != "001" {
			t.Error("Expected 001, got", cf.Tag)
		}
	})
	t.Run("Multiple control fields", func(t *testing.T) {
		cfs := r.ControlField("001", "003")
		if len(cfs) != 2 {
			t.Error("Exected 2, got", len(cfs))
		}
	})
	t.Run("Specific data field", func(t *testing.T) {
		df := r.DataField("245")[0]
		if df.Tag != "245" {
			t.Error("Expected 245, got", df.Tag)
		}
	})
	t.Run("Multiple data fields", func(t *testing.T) {
		dfs := r.DataField("650", "700")
		if len(dfs) != 6 {
			t.Error("Expected 6, got", len(dfs))
		}
	})
	t.Run("Specific subfield", func(t *testing.T) {
		df := r.DataField("260")[0]
		sf := df.SubField("a")[0]
		if sf.Value != "San Diego :" {
			t.Error("Expected San Diego :, got", sf.Value)
		}
	})
	t.Run("Multiple subfields", func(t *testing.T) {
		df := r.DataField("260")[0]
		sfs := df.SubField("a", "b", "z")
		if len(sfs) != 2 {
			t.Error("Expected 2, got", len(sfs))
		}
	})
	t.Run("Filter", func(t *testing.T) {
		sfs := r.Filter("260ac", "245a", "100")
		if len(sfs) != 3 {
			t.Error("Expected 3, got", len(sfs))
		}
		cat := strings.Join(sfs[0], " ")
		if cat != "San Diego : c1993." {
			t.Error("Expected San Diego : c1993., got", cat)
		}
	})
	t.Run("Filter control and data field", func(t *testing.T) {
		sfs := r.Filter("001", "700e")
		if len(sfs) != 2 {
			t.Error("Expected 2, got", len(sfs))
		}
		cat := strings.Join(sfs[0], " ")
		cat += " " + strings.Join(sfs[1], " ")
		if cat != "   92005291  ill." {
			t.Error("Expected    92005291  ill., got", cat)
		}
	})
	t.Run("Filter multiples", func(t *testing.T) {
		sfs := r.Filter("650x")
		if len(sfs) != 2 {
			t.Error("Expected 2, got", len(sfs))
		}
		cat := strings.Join(sfs[0], " ")
		cat += " " + strings.Join(sfs[1], " ")
		if cat != "Juvenile poetry. Poetry." {
			t.Error("Expected Juvenile poetry. Poetry., got", cat)
		}
	})
	t.Run("Filter indicators", func(t *testing.T) {
		sfs := r.Filter("650|*0|x")
		if len(sfs) != 1 {
			t.Error("Expected 1, got", len(sfs))
		}
		cat := strings.Join(sfs[0], " ")
		if cat != "Juvenile poetry." {
			t.Error("Expected Juvenile poetry., got", cat)
		}
	})
}
