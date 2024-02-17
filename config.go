package main
import (
	"errors"
	"encoding/json"
	"os"
)

type Config struct {
	TFTPDirectory	string			`json:"tftp_directory"`
	HTTPPort	string			`json:"http_port"`
	SSHSerPort	string			`json:"ssh_serial_tunnel_port"`
	SSHMonPort	string			`json:"ssh_monitor_port"`
	SerialConfig	struct {
		Port		string		`json:"port"`
		BaudRate	int		`json:"baud_rate"`
	}					`json:"serial_config"`
        Monitor		map[string] string	`json:"monitor"`
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
