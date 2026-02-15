package main

import (
	"encoding/json"
	"io/ioutil"
)

type SSHCFG struct {
	Port            string `json:"port"`
	IdentitFn       string `json:"identity_fn"`
	Authorized_keys string `json:"authorized_keys"`
}

type CalendarConfig struct {
	Enable bool   `json:"enable"`
	Credfn string `json:"credfn"`
}

type NetServ struct {
	SyslogPort    string `json:"syslog_port"`
	LogFile       string `json:"logfile"`
	TFTPDirectory string `json:"tftp_directory"`
	HTTPPort      string `json:"http_port"`
}

type RouterD struct {
	SerialChans  int `json:"serial_chans"`
	MonitorChans int `json:"monitor_chans"`
	SerialMain   int `json:"serial_main"`
	MonitorMain  int `json:"monitor_main"`
}

type SerialD struct {
	Port     string `json:"port"`
	BaudRate int    `json:"baud_rate"`
}

type Config struct {
	NetServices  NetServ           `json:"netservices"`
	SSHSerTun    SSHCFG            `json:"ssh_serial_tunnel"`
	SSHMon       SSHCFG            `json:"ssh_monitor"`
	Router       RouterD           `json:"router"`
	SerialConfig SerialD           `json:"serial_config"`
	Monitor      map[string]string `json:"monitor"`
	Calendar     CalendarConfig    `json:"calendar"`
}

func fetch_config(fn string, key string) (Config, error) {
	var config Config
	var err error
	var fileContent []byte

	if key != "" {
		fileContent, err = DecryptConfig(fn, key)
	} else {
		fileContent, err = ioutil.ReadFile(fn)
	}
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(fileContent, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func enc_config(fn string, key string) error {
	b, err := EncryptConfig(fn, key)
	if err != nil {
		return err
	}
	err = WriteFile("config.rsa", b)
	if err != nil {
		return err
	}
	return nil
}
