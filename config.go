package main
import (
	"errors"
	"encoding/json"
	"os"
)

type SSHCFG struct {
                Port            string          `json:"port"`
                IdentitFn       string          `json:"identity_fn"`
                Authorized_keys string          `json:"authorized_keys"`
        }

type Config struct {
	TFTPDirectory	string			`json:"tftp_directory"`
	HTTPPort	string			`json:"http_port"`
	SSHSerTun	SSHCFG			`json:"ssh_serial_tunnel"`
	SSHMon		SSHCFG			`json:"ssh_monitor"`
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
