package compiler

import (
	"github.com/xorcare/pointer"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing_system/common"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
	"text/template"
)

type Compiler struct {
	Languages map[string]*Language
}

type Config struct {
	DefaultLimits *sandbox.RunConfig   `yaml:"DefaultLimits"`
	Languages     map[string]*Language `yaml:"Languages"`
}

func NewCompiler(ts *common.TestingSystem) *Compiler {
	configPath := filepath.Join(ts.Config.Invoker.CompilerConfigsFolder, "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		logger.Panic("Can not read compiler config at path %s, error: %s", configPath, err.Error())
	}

	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		logger.Panic("Can not parse compiler config, error: %s", err.Error())
	}

	err = config.DefaultLimits.FillIn()
	if err != nil {
		logger.Panic("Can not parse default limits for compilation, error: %s", err.Error())
	}

	c := &Compiler{
		Languages: make(map[string]*Language),
	}
	for name, l := range config.Languages {
		l.Name = name
		if l.Limits == nil {
			l.Limits = config.DefaultLimits
		} else {
			err = l.Limits.FillIn()
			if err != nil {
				logger.Panic("Can not parse limits for compilation of %s, error: %s", name, err.Error())
			}
		}
		if l.TemplateName == nil {
			l.TemplateName = pointer.String(name + ".sh.tmpl")
		}
		l.Template, err = template.ParseFiles(filepath.Join(ts.Config.Invoker.CompilerConfigsFolder, "scripts", *l.TemplateName))
		if err != nil {
			logger.Panic("Can not parse script template for compilation of %s, error: %s", name, err.Error())
		}
		c.Languages[name] = l
	}
	logger.Info("Configured invoker compiler")
	return c
}
