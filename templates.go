package main

import "text/template"

var defaultTemplate = template.Must(template.New("default").Parse(`
if {{ .Field }} == {{ .Zero }} {
	{{ .Field }} = {{ .Default }}
}
`))

type defaultTemplateInput struct {
	Field, Zero, Default string
}

var opTemplate = template.Must(template.New("op").Parse(`
if {{ .Field }} {{ .Op }} {{ .Value }} {
	return errors.New("{{ .Error }}")
}
`))

type opTemplateInput struct {
	Field, Op, Value, Error string
}
