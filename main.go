package main

import (
	"log"
	"go/ast"
	"go/token"
	"go/parser"
	"reflect"
	"strings"
	"path/filepath"
)

type packageDecl struct {
	Name string
	Dir string
	File string
	Structs []structDecl
}

type structDecl struct {
	Name string
	Fields []fieldDecl
}

type fieldDecl struct {
	Name string
	Type string
	Tag string
}

func ReadFile(filename string) packageDecl {

	fset := token.NewFileSet()
	f, e := parser.ParseFile(fset, filename, nil, 0)
	if e != nil {
		log.Fatal(e)
	}

	pd := packageDecl{
		Name: f.Name.Name,
		Dir: filepath.Dir(filename),
		File: filepath.Base(filename),
		Structs: []structDecl{},
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.TypeSpec:
			sd := structDecl{
				Name: n.Name.Name,
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
						fd := fieldDecl{
							Name: fieldName,
							Type: fieldType,
							Tag: structTag.Get("arg"),
						}
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


