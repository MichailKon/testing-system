package config

import "github.com/xorcare/pointer"

type InvokerConfig struct {
	Instances int  `yaml:"Instances"`
	QueueSize *int `yaml:"QueueSize,omitempty"` // default is 10

	CacheSize uint64 `yaml:"CacheSize"`
	CachePath string `yaml:"CachePath"`
}

func fillInInvokerConfig(config *InvokerConfig) {
	if config.QueueSize == nil {
		config.QueueSize = pointer.Int(10)
	}
	if len(config.CachePath) == 0 {
		panic("No invoker cache path specified")
	}
	if config.CacheSize == 0 {
		panic("No invoker cache size specified")
	}
}
