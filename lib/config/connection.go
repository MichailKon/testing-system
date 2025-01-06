package config

type Connection struct {
	Address string `yaml:"Address"`
	// TODO: Add authentification
}

func fillInConnections(config *Config) {
	// TODO: Add auto connection creation if master or storage are set up
}
