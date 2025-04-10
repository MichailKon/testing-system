package config

type InvokerConfig struct {
	Threads   uint64 `yaml:"Threads"`
	Sandboxes uint64 `yaml:"Sandboxes"`
	QueueSize uint64 `yaml:"QueueSize"`

	CacheSize uint64 `yaml:"CacheSize"`
	CachePath string `yaml:"CachePath"`

	SaveOutputHead *uint64 `yaml:"SaveOutputHead,omitempty"`

	CompilerConfigsFolder string `yaml:"CompilerConfigsFolder"`

	CheckerLimits *RunConfig `yaml:"CheckerLimits,omitempty"`
}

func fillInInvokerConfig(config *InvokerConfig) {
	if config.Threads == 0 {
		config.Threads = 1
	}
	if config.Sandboxes == 0 {
		// TODO: find out what number is the best
		config.Sandboxes = 1
	}
	if config.QueueSize == 0 {
		config.QueueSize = 10
	}
	if len(config.CachePath) == 0 {
		panic("No invoker cache path specified")
	}
	if config.CacheSize == 0 {
		panic("No invoker cache size specified")
	}
	if len(config.CompilerConfigsFolder) == 0 {
		panic("No invoker compiler folder specified")
	}

	if config.CheckerLimits == nil {
		config.CheckerLimits = &RunConfig{}
	}
	fillInDefaultCheckerRunConfig(config.CheckerLimits)
}
