package config


// tasks to do define the two stucts we just disccused
// we need it outside in main thus public 
type Config struct {
    Port      int             `yaml:"port"`
    Backends  []BackendConfig `yaml:"backends"`
    Algorithm string          `yaml:"algorithm"`
    HealthInterval  int       `yaml:"health_interval"`  // seconds between checks
    HealthTimeout   int       `yaml:"health_timeout"`   // seconds before giving up
    RateLimit  int `yaml:"rate_limit"`   // number of request per sec
    RateBurst  int `yaml:"rate_burst"`   // burst size
}

type BackendConfig struct {
    URL    string `yaml:"url"`
    Weight int    `yaml:"weight"`
}
