package config

import "time"

type MasterConfig struct {
	InvokersPingInterval time.Duration `yaml:"InvokersPingInterval"`
	SendJobInterval      time.Duration `yaml:"FetchJobInterval"`
	LostJobTimeout       time.Duration `yaml:"LostJobTimeout"`
}

func fillInMasterConfig(config *MasterConfig) {
	if config.InvokersPingInterval == 0 {
		config.InvokersPingInterval = 10 * time.Second
	}
	if config.SendJobInterval == 0 {
		config.SendJobInterval = time.Second
	}
	if config.LostJobTimeout == 0 {
		config.LostJobTimeout = 5 * time.Second
	}
}
