package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port int
}

func Load() Config {
	port := 7012
	if p := strings.TrimSpace(os.Getenv("OFFICEMAN_PORT")); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			port = v
		}
	}
	return Config{Port: port}
}
