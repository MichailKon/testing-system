package tsapiconfig

import "time"

const (
	DefaultLoadFilesHead = 500
)

type Config struct {
	LoadFilesHead        int64         `yaml:"LoadFilesHead"`
	StatusUpdateInterval time.Duration `yaml:"StatusUpdateInterval"`
}
