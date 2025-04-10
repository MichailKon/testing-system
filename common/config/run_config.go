package config

import "testing_system/lib/customfields"

type RunConfig struct {
	TimeLimit     customfields.TimeLimit   `yaml:"TimeLimit" json:"TimeLimit"`
	MemoryLimit   customfields.MemoryLimit `yaml:"MemoryLimit" json:"MemoryLimit"`
	WallTimeLimit customfields.TimeLimit   `yaml:"WallTimeLimit" json:"WallTimeLimit"`

	MaxThreads    uint64                   `yaml:"MaxThreads" json:"MaxThreads"`
	MaxOpenFiles  uint64                   `yaml:"MaxOpenFiles" json:"MaxOpenFiles"`
	MaxOutputSize customfields.MemoryLimit `yaml:"MaxOutputSize" json:"MaxOutputSize"`
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
