package config


// tasks to do define the two stucts we just disccused
// we need it outside in main thus public 
type Config struct {
    Port      int             `yaml:"port"`
    Backends  []BackendConfig `yaml:"backends"`
    Algorithm string          `yaml:"algorithm"`
}

type BackendConfig struct {
    URL    string `yaml:"url"`
    Weight int    `yaml:"weight"`
}
