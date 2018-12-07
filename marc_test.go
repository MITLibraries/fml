package fml

import (
	"os"
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
}
