package config

type DBConfig struct {
	Dsn string `yaml:"Dsn"`

	// InMemory should be used only for tests
	InMemory bool `yaml:"InMemory"`
}
