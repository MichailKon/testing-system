package compiler

import (
	"bytes"
	"fmt"
	"maps"
	"testing_system/invoker/sandbox"
	"text/template"
)

type Language struct {
	Name string `yaml:"-"`

	TemplateValues map[string]interface{} `yaml:"TemplateValues"`

	TemplateName *string                `yaml:"Template,omitempty"`
	Limits       *sandbox.ExecuteConfig `yaml:"Limits,omitempty"`

	Template *template.Template `yaml:"-"`
}

func (l *Language) GenerateScript(source string, binary string) ([]byte, error) {
	var script bytes.Buffer
	values := map[string]interface{}{
		"source": source,
		"binary": binary,
	}
	maps.Copy(values, l.TemplateValues)

	err := l.Template.Execute(&script, values)
	if err != nil {
		return nil, fmt.Errorf("error while creating compile script for language %s, error: %s", l.Name, err.Error())
	}
	return script.Bytes(), nil
}

func (l *Language) GenerateExecuteConfig(stdout *bytes.Buffer) *sandbox.ExecuteConfig {
	c := *l.Limits
	c.Stdout = &sandbox.IORedirect{Output: stdout}
	c.Stderr = &sandbox.IORedirect{Output: stdout}
	return &c
}

func fillInCompileExecuteConfig(c *sandbox.ExecuteConfig) {
	if c.TimeLimit == 0 {
		c.TimeLimit.FromStr("5s")
	}
	if c.WallTimeLimit == 0 {
		c.WallTimeLimit.FromStr("15s")
	}
	if c.MemoryLimit == 0 {
		c.MemoryLimit.FromStr("1g")
	}
	if c.MaxOpenFiles == 0 {
		c.MaxOpenFiles = 64
	}
	if c.MaxThreads == 0 {
		c.MaxThreads = -1
	}
	if c.MaxOutputSize == 0 {
		c.MaxOutputSize.FromStr("1g")
	}
}
