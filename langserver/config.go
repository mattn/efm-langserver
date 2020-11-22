package langserver

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig load configuration from file
func LoadConfig(yamlfile string) (*Config, error) {
	var config = Config{
		ProvideDefinition: true, // Enabled by default.
		Commands:          &[]Command{},
		Languages:         &map[string][]Language{},
		RootMarkers:       &[]string{},
	}
	var config1 Config1

	f, err := os.Open(yamlfile)
	if err != nil {
		log.Println("efm-langserver: no configuration file")
		return &config, nil
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&config1)
	if err != nil || config1.Version == 2 {
		f, err = os.Open(yamlfile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		err = yaml.NewDecoder(f).Decode(&config)
		if err != nil {
			return nil, fmt.Errorf("can not read configuration: %v", err)
		}
	} else {
		config.Version = config1.Version
		config.Commands = &config1.Commands
		config.Logger = config1.Logger
		languages := make(map[string][]Language)
		for k, v := range config1.Languages {
			languages[k] = []Language{v}
		}
		config.Languages = &languages
	}
	config.Filename = yamlfile
	return &config, nil
}
