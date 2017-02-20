package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"
)

type (
	generator struct {
		Package packageDecl
		Buf     *bytes.Buffer
	}

	packageDecl struct {
		Name    string
		Dir     string
		File    string
		Structs []structDecl
	}

	structDecl struct {
		Name   string
		Fields []fieldDecl
	}

	fieldDecl struct {
		Name  string
		Type  string
		Tag   string
		Conds []condDecl
	}

	condDecl struct {
		Name  string
		Value string
	}
)

// var for flag
var (
	typeNames    = flag.String("type", "", "comma-separated list of type names; must be set")
	outputOption = flag.String("out", "", "output file name; default srcdir/<input_file_name>_argumenter.go")
)

// var for template
var (
	packageTemplate = template.Must(template.New("package").Parse(`
// Code generated by "argumenter -type {{ .Types }}"; DO NOT EDIT
package {{ .Name }}

import "errors"
`))

	funcHeaderTemplate = template.Must(template.New("func_header").Parse(`
func ({{ .Self }} *{{ .Name }}) Valid() error {
`))

	funcFooterTemplate = template.Must(template.New("func_footer").Parse(`
	return nil
}
`))

	defaultTemplate = template.Must(template.New("default").Parse(`
if {{ .Field }} == {{ .Zero }} {
	{{ .Field }} = {{ .Default }}
}
`))

	opTemplate = template.Must(template.New("op").Parse(`
if {{ .Field }} {{ .Op }} {{ .Value }} {
	return errors.New("{{ .Error }}")
}
`))
)

// type for template
type (
	packageTemplateInput struct {
		Name, Types string
	}

	funcHeaderTemplateInput struct {
		Self, Name string
	}

	defaultTemplateInput struct {
		Field, Zero, Default string
	}

	opTemplateInput struct {
		Field, Op, Value, Error string
	}
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "\tstringer [flags] file\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()
	flag.Usage = usage
	filename := flag.Arg(0)
	if filename == "" || *typeNames == "" {
		flag.Usage()
		os.Exit(2)
	}
	g := newGenerator()
	g.ReadFile(filename)

	src, e := g.Generate(strings.Split(*typeNames, ","))
	if e != nil {
		panic(e)
	}

	var output string
	if *outputOption == "" {
		output = g.Package.Dir + "/" + strings.TrimRight(g.Package.File, ".go") + "_argumenter.go"
	} else {
		output = *outputOption
	}

	err := ioutil.WriteFile(output, src, 0644)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}
}

func newGenerator() *generator {
	return &generator{Buf: new(bytes.Buffer)}
}

func newFieldDecl(name, typ, tag string) fieldDecl {
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

func (g *generator) ReadFile(filename string) {

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
			if n.Name.Obj != nil && n.Name.Obj.Kind == ast.Typ {
				ast.Inspect(n, func(nn ast.Node) bool {
					switch nn := nn.(type) {
					case *ast.Field:
						fieldName := nn.Names[0].Name
						fieldType := types.ExprString(nn.Type)
						fieldTag := nn.Tag.Value
						structTag := reflect.StructTag(strings.Trim(fieldTag, "`"))
						fd := newFieldDecl(fieldName, fieldType, structTag.Get("arg"))
						sd.Fields = append(sd.Fields, fd)
					}
					return true
				})
			}
			pd.Structs = append(pd.Structs, sd)
		}
		return true
	})

	g.Package = pd
}

func (g *generator) Generate(typeNames []string) ([]byte, error) {
	e := packageTemplate.Execute(g.Buf, packageTemplateInput{g.Package.Name, strings.Join(typeNames, ",")})
	if e != nil {
		return nil, e
	}

	for _, s := range g.SelectStructs(typeNames) {
		e := s.Generate(g.Buf)
		if e != nil {
			return nil, e
		}
	}

	src, e := format.Source(g.Buf.Bytes())
	if e != nil {
		return nil, e
	}
	return src, nil
}

func (g *generator) SelectStructs(names []string) []structDecl {
	result := []structDecl{}
	for _, s := range g.Package.Structs {
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
	zerostr := strings.Replace(zero, `"`, `\"`, -1)
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
				field, "==", zero, fmt.Sprintf("%s must not %s", f.Name, zerostr),
			})
		case "zero":
			e = opTemplate.Execute(w, opTemplateInput{
				field, "!=", zero, fmt.Sprintf("%s must %s", f.Name, zerostr),
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
