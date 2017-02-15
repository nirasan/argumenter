package main

import "text/template"

var defaultTemplate = template.Must(template.New("default").Parse(`
if {{ .Self }}.{{ .Field }} == {{ .Zero }} {
	{{ .Self }}.{{ .Field }} = {{ .Default }}
}
`))

type defaultTemplateInput struct {
	Self, Field, Zero, Default string
}

var opTemplate = template.Must(template.New("op").Parse(`
if {{ .Self }}.{{ .Field }} {{ .Op }} {{ .Value }} {
	return errors.New("{{ .Error }}")
}
`))

type opTemplateInput struct {
	Self, Field, Op, Value, Error string
}
