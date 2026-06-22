// internal/parser/parser.go
package parser

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
    "github.com/pranavbhole123/load-balancer/internal/config"
)

func Load(path string) (*config.Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("parser: reading config file %s: %w", path, err)
    }

    var con config.Config
    err = yaml.Unmarshal(data, &con)
    if err != nil {
        return nil, fmt.Errorf("parser: unmarshaling config: %w", err)
    }

    if con.Port == 0 {
        return nil, fmt.Errorf("parser: port is required")
    }
    if len(con.Backends) == 0 {
        return nil, fmt.Errorf("parser: at least one backend is required")
    }
    for i, b := range con.Backends {
        if b.URL == "" {
            return nil, fmt.Errorf("parser: backend[%d] url is required", i)
        }
    }

    if con.HealthInterval == 0 {
        con.HealthInterval = 10
    }  
    if con.HealthTimeout == 0 {
        con.HealthTimeout = 3
    
    if con.HealthTimeout >= con.HealthInterval {
    return nil, fmt.Errorf("parser: health_timeout (%ds) must be less than health_interval (%ds)", 
        con.HealthTimeout, con.HealthInterval)
    }}

    return &con, nil
}