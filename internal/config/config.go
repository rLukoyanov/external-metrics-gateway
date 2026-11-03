package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Route struct {
	Prefix      string `yaml:"prefix"`
	Source      string `yaml:"source"`
	StripPrefix bool   `yaml:"stripPrefix"`
}

type Config struct {
	ListenAddr string `yaml:"listenAddr"`
	Prometheus struct {
		URL string `yaml:"url"`
	} `yaml:"prometheus"`
	KEDA struct {
		URL string `yaml:"url"`
	} `yaml:"keda"`
	Routes        []Route `yaml:"routes"`
	DefaultSource string  `yaml:"defaultSource"`
}

func Load(path string) (Config, error) {
	cfg := Config{
		ListenAddr:    ":8080",
		DefaultSource: "prometheus",
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8080"
	}
	if cfg.DefaultSource == "" {
		cfg.DefaultSource = "prometheus"
	}
	return cfg, nil
}
