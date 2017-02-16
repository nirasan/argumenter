package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"path/filepath"
	"reflect"
	"strings"
)

type packageDecl struct {
	Name    string
	Dir     string
	File    string
	Structs []structDecl
}

type structDecl struct {
	Name   string
	Fields []fieldDecl
}

type fieldDecl struct {
	Name  string
	Type  string
	Tag   string
	Conds []condDecl
}

type condDecl struct {
	Name  string
	Value string
}

func ReadFile(filename string) packageDecl {

	fset := token.NewFileSet()
	f, e := parser.ParseFile(fset, filename, nil, 0)
	if e != nil {
		log.Fatal(e)
	}

	pd := packageDecl{
		Name:    f.Name.Name,
		Dir:     filepath.Dir(filename),
		File:    filepath.Base(filename),
		Structs: []structDecl{},
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.TypeSpec:
			if _, ok := n.Type.(*ast.StructType); !ok {
				return true
			}
			sd := structDecl{
				Name:   n.Name.Name,
				Fields: []fieldDecl{},
			}
			// ast.Print(fset, n)
			if n.Name.Obj != nil && n.Name.Obj.Kind == ast.Typ {
				ast.Inspect(n, func(nn ast.Node) bool {
					switch nn := nn.(type) {
					case *ast.Field:
						fieldName := nn.Names[0].Name
						fieldType := nn.Type.(*ast.Ident).Name
						fieldTag := nn.Tag.Value
						structTag := reflect.StructTag(strings.Trim(fieldTag, "`"))
						fd := NewFieldDecl(fieldName, fieldType, structTag.Get("arg"))
						sd.Fields = append(sd.Fields, fd)
					}
					return true
				})
			}
			pd.Structs = append(pd.Structs, sd)
		}
		return true
	})

	return pd
}

func NewFieldDecl(name, typ, tag string) fieldDecl {
	f := fieldDecl{
		Name:  name,
		Type:  typ,
		Tag:   tag,
		Conds: []condDecl{},
	}
	for _, t := range strings.Split(tag, ",") {
		pair := strings.SplitN(t, "=", 2)
		var c condDecl
		if len(pair) == 2 {
			c = condDecl{pair[0], pair[1]}
		} else if len(pair) == 1 {
			c = condDecl{pair[0], ""}
		}
		f.Conds = append(f.Conds, c)
	}
	return f
}

func (p packageDecl) SelectStructs(names []string) []structDecl {
	result := []structDecl{}
	for _, s := range p.Structs {
		for _, n := range names {
			if s.Name == n {
				result = append(result, s)
			}
		}
	}
	return result
}

func (s structDecl) Generate(w io.Writer) error {
	self := strings.ToLower(s.Name)
	self = string(self[:1])

	e := funcHeaderTemplate.Execute(w, funcHeaderTemplateInput{self, s.Name})
	if e != nil {
		return e
	}

	for _, f := range s.Fields {
		e := f.Generate(w, self)
		if e != nil {
			return e
		}
	}

	e = funcFooterTemplate.Execute(w, nil)
	if e != nil {
		return e
	}

	return nil
}

func (f fieldDecl) Generate(w io.Writer, self string) error {
	zero := f.Zero()
	field := self + "." + f.Name
	fieldlen := "len(" + field + ")"
	for _, c := range f.Conds {
		var e error
		switch c.Name {
		case "default":
			var v string
			if f.Type == "string" {
				v = fmt.Sprintf(`"%s"`, c.Value)
			} else {
				v = c.Value
			}
			e = defaultTemplate.Execute(w, defaultTemplateInput{
				field, zero, v,
			})
		case "required", "notzero":
			e = opTemplate.Execute(w, opTemplateInput{
				field, "==", zero, fmt.Sprintf("%s must not %s", f.Name, zero),
			})
		case "zero":
			e = opTemplate.Execute(w, opTemplateInput{
				field, "!=", zero, fmt.Sprintf("%s must %s", f.Name, zero),
			})
		case "min", "gte":
			if f.IsNumber() {
				e = opTemplate.Execute(w, opTemplateInput{
					field, "<", c.Value, fmt.Sprintf("%s must greater than or equal %s", f.Name, c.Value),
				})
			}
		case "max", "lte":
			if f.IsNumber() {
				e = opTemplate.Execute(w, opTemplateInput{
					field, ">", c.Value, fmt.Sprintf("%s must less than or equal %s", f.Name, c.Value),
				})
			}
		case "gt":
			if f.IsNumber() {
				e = opTemplate.Execute(w, opTemplateInput{
					field, "<=", c.Value, fmt.Sprintf("%s must greater than %s", f.Name, c.Value),
				})
			}
		case "lt":
			if f.IsNumber() {
				e = opTemplate.Execute(w, opTemplateInput{
					field, ">=", c.Value, fmt.Sprintf("%s must less than %s", f.Name, c.Value),
				})
			}
		case "len":
			if f.IsSlice() {
				e = opTemplate.Execute(w, opTemplateInput{
					fieldlen, "!=", c.Value, fmt.Sprintf("%s length must %s", f.Name, c.Value),
				})
			}

		case "lenmin":
			if f.IsSlice() {
				e = opTemplate.Execute(w, opTemplateInput{
					fieldlen, "<", c.Value, fmt.Sprintf("%s length must greater than or equal %s", f.Name, c.Value),
				})
			}
		case "lenmax":
			if f.IsSlice() {
				e = opTemplate.Execute(w, opTemplateInput{
					fieldlen, ">", c.Value, fmt.Sprintf("%s length must less than or equal %s", f.Name, c.Value),
				})
			}
		}
		if e != nil {
			return e
		}
	}
	return nil
}

func (f fieldDecl) Zero() string {
	var zero string
	switch {
	case f.IsNumber():
		zero = "0"
	case f.IsBool():
		zero = "false"
	case f.IsString():
		zero = `""`
	case f.IsSlice() || f.IsMap() || f.IsChan() || f.IsFunc() || f.IsInterface() || f.IsPtr():
		zero = "nil"
	default:
		zero = fmt.Sprintf("*new(%s)", f.Type)
	}
	return zero
}

func (f fieldDecl) IsNumber() bool {
	return f.IsInt() || f.IsUint() || f.IsFloat() || f.IsComplex()
}

func (f fieldDecl) IsInt() bool {
	if start(f.Type, "interface") {
		return false
	}
	return start(f.Type, "int") || start(f.Type, "byte")
}

func (f fieldDecl) IsUint() bool {
	return start(f.Type, "uint") || start(f.Type, "rune")
}

func (f fieldDecl) IsFloat() bool {
	return start(f.Type, "float")
}

func (f fieldDecl) IsComplex() bool {
	return start(f.Type, "complex")
}

func (f fieldDecl) IsBool() bool {
	return start(f.Type, "bool")
}

func (f fieldDecl) IsString() bool {
	return start(f.Type, "string")
}

func (f fieldDecl) IsMap() bool {
	return start(f.Type, "map")
}

func (f fieldDecl) IsSlice() bool {
	return start(f.Type, "[")
}

func (f fieldDecl) IsPtr() bool {
	return start(f.Type, "*")
}

func (f fieldDecl) IsFunc() bool {
	return start(f.Type, "func")
}

func (f fieldDecl) IsChan() bool {
	return start(f.Type, "chan") || start(f.Type, "<-chan")
}

func (f fieldDecl) IsInterface() bool {
	return start(f.Type, "interface")
}

func start(s, needle string) bool {
	return strings.Index(s, needle) == 0
}
