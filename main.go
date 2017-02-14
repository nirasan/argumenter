package main

import (
	"log"
	"go/ast"
	"go/token"
	"go/parser"
	"reflect"
	"strings"
)

type structDecl struct {
	Fields []fieldDecl
}

type fieldDecl struct {
	Name string
	Type string
	Tag string
}

func ReadFile(filename string) {
	fset := token.NewFileSet()
	f, e := parser.ParseFile(fset, filename, nil, 0)
	if e != nil {
		log.Fatal(e)
	}

	ast.Inspect(f, func(n ast.Node) bool {
		//x, ok := n.(*ast.Ident)
		//if !ok || x.Obj == nil || x.Obj.Kind != ast.Typ {
		//	return true
		//}
		//ast.Print(fset, x)
		//ast.Inspect(x, func(nn ast.Node) bool {
		//	xx, ok := nn.(*ast.Field)
		//	if !ok {
		//		//
		//		// return true
		//	}
		//	ast.Print(fset, xx)
		//	return true
		//})
		//return true

		switch n := n.(type) {
		case *ast.TypeSpec:
			structName := n.Name.Name
			log.Printf("struct name: %s", structName)
			// ast.Print(fset, n)
			if n.Name.Obj != nil && n.Name.Obj.Kind == ast.Typ {
				ast.Inspect(n, func(nn ast.Node) bool {
					switch nn := nn.(type) {
					case *ast.Field:
						fieldName := nn.Names[0].Name
						fieldType := nn.Type.(*ast.Ident).Name
						fieldTag := nn.Tag.Value
						structTag := reflect.StructTag(strings.Trim(fieldTag, "`"))
						log.Printf("field name: %s, type: %s, tag: %s, st: %s, arg: %s", fieldName, fieldType, fieldTag, structTag, structTag.Get("arg"))
						// ast.Print(fset, nn)
					}
					return true
				})
			}
		}
		return true
	})
}


