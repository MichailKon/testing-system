package config

import "testing_system/lib/customfields"

type RunConfig struct {
	TimeLimit     customfields.Time   `yaml:"TimeLimit" json:"TimeLimit"`
	MemoryLimit   customfields.Memory `yaml:"MemoryLimit" json:"MemoryLimit"`
	WallTimeLimit customfields.Time   `yaml:"WallTimeLimit" json:"WallTimeLimit"`

	// MaxThreads specifies max number of threads and/or processes.
	// If MaxThreads is unspecified (equals 0) it is considered as 1.
	// If MaxThreads = -1, any number of threads is allowed
	MaxThreads int64 `yaml:"MaxThreads" json:"MaxThreads"`

	// MaxOpenFiles specifies max number of files, opened by process
	// If MaxOpenFiles is unspecified (equals 0), it is considered as 64
	MaxOpenFiles uint64 `yaml:"MaxOpenFiles" json:"MaxOpenFiles"`

	// MaxOutputSize specifies max output in EACH file.
	// If MaxOpenFiles is unspecified (equals 0), it is considered as 1g
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
	if config.MaxOpenFiles == 0 {
		config.MaxOpenFiles = 64
	}
	if config.MaxOutputSize == 0 {
		config.MaxOutputSize.FromStr("1g")
	}
}
