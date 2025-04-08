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
	DefaultLimits *sandbox.ExecuteConfig `yaml:"DefaultLimits"`
	Languages     map[string]*Language   `yaml:"Languages"`
}

func NewCompiler(ts *common.TestingSystem) *Compiler {
	configPath := filepath.Join(ts.Config.Invoker.CompilerConfigsFolder, "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		logger.Panic("Can not read compiler config at path %s, error: %s", configPath, err.Error())
	}

	var languageConfig Config
	err = yaml.Unmarshal(configData, &languageConfig)
	if err != nil {
		logger.Panic("Can not parse compiler config, error: %s", err.Error())
	}

	if languageConfig.DefaultLimits == nil {
		languageConfig.DefaultLimits = &sandbox.ExecuteConfig{}
	}
	fillInCompileExecuteConfig(languageConfig.DefaultLimits)

	c := &Compiler{
		Languages: make(map[string]*Language),
	}
	for name, l := range languageConfig.Languages {
		l.Name = name
		if l.Limits == nil {
			l.Limits = languageConfig.DefaultLimits
		} else {
			fillInCompileExecuteConfig(l.Limits)
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
