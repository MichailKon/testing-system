package config

import "testing_system/lib/customfields"

type RunConfig struct {
	TimeLimit     customfields.Time   `yaml:"TimeLimit" json:"TimeLimit"`
	MemoryLimit   customfields.Memory `yaml:"MemoryLimit" json:"MemoryLimit"`
	WallTimeLimit customfields.Time   `yaml:"WallTimeLimit" json:"WallTimeLimit"`

	MaxThreads    uint64              `yaml:"MaxThreads" json:"MaxThreads"`
	MaxOpenFiles  uint64              `yaml:"MaxOpenFiles" json:"MaxOpenFiles"`
	MaxOutputSize customfields.Memory `yaml:"MaxOutputSize" json:"MaxOutputSize"`
}

func fillInDefaultCheckerRunConfig(config *RunConfig) {
	if config.TimeLimit == 0 {
		config.TimeLimit.FromStr("15s")
	}
	if config.MemoryLimit == 0 {
		config.MemoryLimit.FromStr("1g")
	}
	if config.WallTimeLimit == 0 {
		config.WallTimeLimit.FromStr("30s")
	}
	if config.MaxThreads == 0 {
		config.MaxThreads = 1
	}
	if config.MaxOpenFiles == 0 {
		config.MaxOpenFiles = 10
	}
	if config.MaxOutputSize == 0 {
		config.MaxOutputSize.FromStr("1g")
	}
}
