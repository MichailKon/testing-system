package tsapiconfig

const (
	DefaultLoadFilesHead = 500
)

type Config struct {
	LoadFilesHead int64 `yaml:"LoadFilesHead"`
}
