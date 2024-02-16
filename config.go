package main
import (
	"errors"
	"encoding/json"
	"os"
)

type Config struct {
	TFTPDirectory string `json:"tftp_directory"`
	HTTPPort      string `json:"http_port"`
	SSHPort       string `json:"ssh_port"`
	SerialConfig  struct {
		Port     string `json:"port"`
		BaudRate int    `json:"baud_rate"`
	} `json:"serial_config"`
}

func fetch_config(fn string) (Config, error) {
	var config Config

	configFile, err := os.Open(fn)
	if err != nil {
		return config,  errors.New("config file not found")
	}
	defer configFile.Close()

	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		return config, errors.New("json error")
	}
	return config, nil
}
