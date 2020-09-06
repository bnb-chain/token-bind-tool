package config

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Config struct {
	ContractData    string `json:"contract_data"`
}

func (bindConfig *Config) validate() error {
	_, err := hex.DecodeString(bindConfig.ContractData)
	if err != nil {
		return fmt.Errorf("invalid contract byte code: %s", err.Error())
	}
	return nil
}

func ReadConfigData(configPath string) (Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %s", err.Error())
	}
	fileData, err := ioutil.ReadAll(file)
	if err != nil {
		return Config{}, err
	}
	var config Config
	err = json.Unmarshal(fileData, &config)
	if err != nil {
		return Config{}, err
	}
	err = config.validate()
	if err != nil {
		return Config{}, err
	}
	return config, nil
}
