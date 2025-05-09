package config

import "time"

type InvokerConfig struct {
	// PublicAddress defines address for public access to invoker from master if the server is set up locally with some proxy
	PublicAddress *string `yaml:"PublicAddress,omitempty"`

	Threads   int `yaml:"Threads"`
	Sandboxes int `yaml:"Sandboxes"`
	QueueSize int `yaml:"QueueSize"`

	SandboxType     string `yaml:"SandboxType"`
	SandboxHomePath string `yaml:"SandboxHomePath"`

	MasterPingInterval time.Duration `yaml:"MasterPingInterval"`

	CacheSize uint64 `yaml:"CacheSize"`
	CachePath string `yaml:"CachePath"`

	SaveOutputHead *uint64 `yaml:"SaveOutputHead,omitempty"`

	CompilerConfigsFolder string `yaml:"CompilerConfigsFolder"`

	CheckerLimits *RunLimitsConfig `yaml:"CheckerLimits,omitempty"`
}

func FillInInvokerConfig(config *InvokerConfig) {
	if config.Threads == 0 {
		config.Threads = 1
	}
	if config.Sandboxes == 0 {
		config.Sandboxes = 2
	}
	if config.QueueSize == 0 {
		config.QueueSize = 2
	}
	if config.SandboxType == "" {
		config.SandboxType = "simple"
	}
	if config.SandboxHomePath == "" {
		switch config.SandboxType {
		case "simple":
			panic("No sandbox home path specified")
		case "isolate":
			panic("No isolate home path specified (it is used for meta files)")
		default:
			panic("unsupported sandbox type: " + config.SandboxType)
		}
	}
	if config.MasterPingInterval == 0 {
		config.MasterPingInterval = time.Second * 10
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
		config.CheckerLimits = &RunLimitsConfig{}
	}
	fillInDefaultCheckerRunLimitsConfig(config.CheckerLimits)
}
