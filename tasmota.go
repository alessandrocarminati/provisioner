package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"io/ioutil"
	"net/http"
)

type DeviceState struct {
	Power string `json:"POWER"`
}

func (c *CmdCtx) tasmotaSwitch(state string) error {
	err := errors.New(fmt.Sprintf("Unknown command: %s", state))

	if ((state=="ON") || (state=="OFF")) {
		tasmota_host, ok := (*(*c).monitor).monitorConfig["tasmota_host"]
		if ok {
			err = TasmotaSetState(tasmota_host, state)
		}
	}
	return err
}

func (c *CmdCtx) TasmotaQueryState(host string) (bool, error) {
	url := fmt.Sprintf("http://%s/cm?cmnd=power", host)
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, errors.New("unable to get device state\r\n")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var deviceState DeviceState
	err = json.Unmarshal(body, &deviceState)
	if err != nil {
		return false, err
	}

	switch deviceState.Power {
	case "ON":
		return true, nil
	case "OFF":
		return false, nil
	default:
		return false, errors.New("unknown device state")
	}
}

func TasmotaSetState(host string, state string) error {
	if state != "ON" && state != "OFF" {
		return errors.New("invalid state provided\r\n")
	}

	url := fmt.Sprintf("http://%s/cm?cmnd=power+%s", host, state)
	resp, err := http.Get(url)
	if err != nil {
		debugPrint(log.Printf, levelError, err.Error() )
		return errors.New("unable to set device state\r\n")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("unable to set device state\r\n")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var deviceState DeviceState
	err = json.Unmarshal(body, &deviceState)
	if err != nil {
		return err
	}

	if deviceState.Power != state {
		return errors.New("device state not set correctly\r\n")
	}

	return nil
}

