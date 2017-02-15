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
	"regexp"
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

func (f fieldDecl) Generate(w io.Writer, self string) error {
	zero := f.Zero()
	for _, c := range f.Conds {
		var e error
		switch c.Name {
		case "default":
			e = defaultTemplate.Execute(w, defaultTemplateInput{
				self, f.Name, zero, c.Value,
			})
		case "required", "notzero":
			e = opTemplate.Execute(w, opTemplateInput{
				self, f.Name, "==", zero, fmt.Sprintf("%s must not %s", f.Name, zero),
			})
		case "zero":
			e = opTemplate.Execute(w, opTemplateInput{
				self, f.Name, "!=", zero, fmt.Sprintf("%s must %s", f.Name, zero),
			})
		case "min", "gte":
			e = opTemplate.Execute(w, opTemplateInput{
				self, f.Name, "<", c.Value, fmt.Sprintf("%s must greater than or equal %s", f.Name, c.Value),
			})
		case "max", "lte":
			e = opTemplate.Execute(w, opTemplateInput{
				self, f.Name, ">", c.Value, fmt.Sprintf("%s must less than or equal %s", f.Name, c.Value),
			})
		case "gt":
			e = opTemplate.Execute(w, opTemplateInput{
				self, f.Name, "<=", c.Value, fmt.Sprintf("%s must greater than %s", f.Name, c.Value),
			})
		case "lt":
			e = opTemplate.Execute(w, opTemplateInput{
				self, f.Name, ">=", c.Value, fmt.Sprintf("%s must less than"+
					" %s", f.Name, c.Value),
			})
		}
		if e != nil {
			return e
		}
	}
	return nil
}

func (f fieldDecl) Zero() string {
	var zero string
	if ok, _ := regexp.MatchString(`^(?:u?int|float)`, f.Type); ok {
		zero = "0"
	} else if ok, _ := regexp.MatchString(`^(?:\[\]|map)`, f.Type); ok {
		zero = "nil"
	} else if f.Type == "bool" {
		zero = "false"
	}
	return zero
}
