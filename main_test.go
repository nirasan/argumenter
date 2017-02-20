package main

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestGenerator_ReadFile(t *testing.T) {
	g := newGenerator()
	g.ReadFile("test/file1.go")
	p := g.Package
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
		Name  string
		Type  string
		Tag   string
		Conds map[string]string
	}{
		{"Name", "string", "required", map[string]string{"required": ""}},
		{"Color", "int64", "required", map[string]string{"required": ""}},
		{"Amount", "uint8", "min=1,max=100,default=1", map[string]string{"min": "1", "max": "100", "default": "1"}},
	}
	for i, sample := range fieldSamples {
		f := s.Fields[i]
		if f.Name != sample.Name || f.Type != sample.Type || f.Tag != sample.Tag {
			t.Errorf("invalid field: %v", f)
		} else {
			t.Logf("valid field: %v", f)
		}
		for _, c := range f.Conds {
			if v, ok := sample.Conds[c.Name]; !ok || c.Value != v {
				t.Errorf("invalid cond: %v", c)
			} else {
				t.Logf("valid cond: %v", c)
			}
		}
	}
}

func TestGenerator_Generate(t *testing.T) {
	g := newGenerator()
	g.ReadFile("test/file1.go")
	src, e := g.Generate([]string{"Pill"})
	if e != nil {
		t.Error(e)
	}
	t.Logf("%s", src)
}

func TestGenerator_SelectStructs(t *testing.T) {
	p := packageDecl{
		Structs: []structDecl{
			structDecl{Name: "Dog"},
			structDecl{Name: "Cat"},
			structDecl{Name: "Bird"},
		},
	}
	g := &generator{Package: p}
	ss := g.SelectStructs([]string{"Cat"})
	if len(ss) != 1 || ss[0].Name != "Cat" {
		t.Error("error", ss)
	}
	ss = g.SelectStructs([]string{"Cat", "Bird"})
	if len(ss) != 2 || ss[0].Name != "Cat" || ss[1].Name != "Bird" {
		t.Error("error", ss)
	}
}

func TestStructDecl_Generate(t *testing.T) {
	s := structDecl{
		Name: "MyStruct",
		Fields: []fieldDecl{
			newFieldDecl("N", "int", "default=1,min=0,max=1"),
			newFieldDecl("S", "string", "required"),
			newFieldDecl("B", "bool", "required"),
		},
	}
	w := new(bytes.Buffer)
	e := s.Generate(w)
	if e != nil {
		t.Error("error: ", e)
	} else {
		t.Logf("%v", w.String())
	}
}

func TestFieldDecl_Generate(t *testing.T) {
	samples := []struct {
		Name, Type, Tag, Out string
	}{
		{"N", "int", "default=1", `if self.N == 0 { self.N = 1 }`},
		{"N", "uint", "default=1", `if self.N == 0 { self.N = 1 }`},
		{"N", "float32", "default=1", `if self.N == 0 { self.N = 1 }`},
		{"B", "bool", "default=true", `if self.B == false { self.B = true }`},
		{"S", "string", "default=hello", `if self.S == "" { self.S = "hello" }`},
		{"F", "func()", "default=func(){}", `if self.F == nil { self.F = func(){} }`},
		{"S", "[]int", "default=[]int{}", `if self.S == nil { self.S = []int{} }`},
		{"M", "map[int]bool", "default=make(map[int]bool)", `if self.M == nil { self.M = make(map[int]bool) }`},
		{"D", "*Dog", "default=&Dog{}", `if self.D == nil { self.D = &Dog{} }`},
		{"I", "interface{}", "default=0", `if self.I == nil { self.I = 0 }`},
		{"A", "[2]int", `default=[2]int{}`, `if self.A == nil { self.A = [2]int{} }`},
		{"List", "[]int", "required", `if self.List == nil { return errors.New("List must not nil") }`},
		{"List", "[]int", "zero", `if self.List != nil { return errors.New("List must nil") }`},
		{"N", "int", "min=0", `if self.N < 0 { return errors.New("N must greater than or equal 0") }`},
		{"N", "int", "max=100", `if self.N > 100 { return errors.New("N must less than or equal 100") }`},
		{"N", "int", "gt=0", `if self.N <= 0 { return errors.New("N must greater than 0") }`},
		{"N", "int", "lt=100", `if self.N >= 100 { return errors.New("N must less than 100") }`},
		{"List", "[]int", "len=2", `if len(self.List) != 2 { return errors.New("List length must 2") }`},
		{"List", "[]int", "lenmin=1", `if len(self.List) < 1 { return errors.New("List length must greater than or equal 1") }`},
		{"List", "[]int", "lenmax=10", `if len(self.List) > 10 { return errors.New("List length must less than or equal 10") }`},
	}
	w := new(bytes.Buffer)
	for _, sample := range samples {
		f := newFieldDecl(sample.Name, sample.Type, sample.Tag)
		e := f.Generate(w, "self")
		if e != nil {
			t.Errorf("error: %v", e)
		}

		re := regexp.MustCompile(`[\s\n]+`)
		out := re.ReplaceAllString(w.String(), " ")
		out = strings.Trim(out, " ")
		if out != sample.Out {
			t.Errorf("not match:\nEXPECT:\t%v\nOUT:\t%v", sample.Out, out)
		} else {
			t.Logf("match: %v", out)
		}
		w.Reset()
	}
}
