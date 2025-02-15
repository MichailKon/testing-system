package compiler

import (
	"bytes"
	"fmt"
	"testing_system/invoker/sandbox"
	"text/template"
)

type Language struct {
	Name string `yaml:"-"`

	TemplateValues map[string]string `yaml:"TemplateValues"`

	TemplateName *string            `yaml:"Template,omitempty"`
	Limits       *sandbox.RunConfig `yaml:"Limits,omitempty"`

	Template *template.Template `yaml:"-"`
}

func (l *Language) GenerateScript(source string, binary string) ([]byte, error) {
	var script bytes.Buffer
	values := map[string]interface{}{
		"source": source,
		"binary": binary,
	}
	for k, v := range l.TemplateValues {
		values[k] = v // Go really does not have another way to do it!
	}

	err := l.Template.Execute(&script, l.TemplateValues)
	if err != nil {
		return nil, fmt.Errorf("error while creating compile script for language %s, error: %s", l.Name, err.Error())
	}
	return script.Bytes(), nil
}
