package clientconfig

import (
	"testing_system/clients/tsapi/tsapiconfig"
	tsconfig "testing_system/common/config"
	"testing_system/lib/logger"
)

type Config struct {
	Address string `yaml:"Address"`

	Logger *logger.Config `yaml:"Logger"`

	StorageConnection *tsconfig.Connection `yaml:"StorageConnection"`
	MasterConnection  *tsconfig.Connection `yaml:"MasterConnection"`
	DB                tsconfig.DBConfig    `yaml:"DB"`

	ResourcesPath string `yaml:"ResourcesPath"`

	TestingSystemAPI *tsapiconfig.Config `yaml:"TestingSystemAPI"`
	Admin            bool                `yaml:"Admin"`
}
