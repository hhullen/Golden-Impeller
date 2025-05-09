package supports

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	envFile = ".env.yaml"
)

func GetEnvCfg() (map[string]string, error) {
	file, err := os.Open(envFile)
	if err != nil {
		return nil, err
	}

	envCfg := make(map[string]string)
	if yaml.NewDecoder(file).Decode(envCfg) != nil {
		return nil, err
	}

	return envCfg, nil
}
