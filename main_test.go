package main

import "testing"

func TestReadFile(t *testing.T) {
	p := ReadFile("test/file1.go")
	if p.Name != "main" || p.Dir != "test" || p.File != "file1.go" || len(p.Structs) != 1 {
		t.Errorf("invalid package: %v", p)
	} else {
		t.Logf("valid package: %v", p)
	}
	s := p.Structs[0]
	if s.Name != "Pill" || len(s.Fields) != 3 {
		t.Errorf("invalid struct: %v", s)
	} else {
		t.Logf("valid struct: %v", s)
	}
	fieldSamples := []struct {
		Name string
		Type string
		Tag string
	}{
		{ "Name", "string", "required" },
		{ "Color", "int64", "required" },
		{ "Amount", "uint8", "min=1,max=100" },
	}
	for i, sample := range fieldSamples {
		f := s.Fields[i]
		if f.Name != sample.Name || f.Type != sample.Type || f.Tag != sample.Tag {
			t.Errorf("invalid field: %v", f)
		} else {
			t.Logf("valid field: %v", f)
		}
	}
}