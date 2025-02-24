package config

type StorageConfig struct {
	StoragePath string `yaml:"StoragePath"`
}

func FillInStorageConfig(config *StorageConfig) {
	if len(config.StoragePath) == 0 {
		panic("No storage path specified")
	}
}
