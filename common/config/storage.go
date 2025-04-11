package config

type StorageConfig struct {
	StoragePath string `yaml:"StoragePath"`

	BlockSize uint `yaml:"BlockSize"`
}

func fillInStorageConfig(config *StorageConfig) {
	if len(config.StoragePath) == 0 {
		panic("No storage path specified")
	}

	if config.BlockSize == 0 {
		config.BlockSize = 3
	}
}
