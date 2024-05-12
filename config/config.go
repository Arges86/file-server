package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
}
type Server struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}
type Database struct {
	File string `yaml:"file"`
}

func NewConfig() (*Config, error) {
	config := &Config{}

	// Open config file
	file, err := os.Open("./config.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}
