package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
)

func (c *CmdCtx) beakerSwitch(state string) error {
	var ignoressl bool

	errstr := "Config error\r\n"
	debugPrint(log.Printf, levelDebug, "fetching beaker data")
	username, ok := (*(*c).monitor).monitorConfig["beaker_username"]
	if !ok {
		return errors.New(errstr)
	}
	password, ok := (*(*c).monitor).monitorConfig["beaker_password"]
	if !ok {
		return errors.New(errstr)
	}
	ignoresslstr, ok := (*(*c).monitor).monitorConfig["beaker_ignoressl"]
	if !ok {
		return errors.New(errstr)
	}
	if ignoresslstr == "true" || ignoresslstr == "false" {
		if ignoresslstr == "true" {
			ignoressl = true
		} else {
			ignoressl = false
		}
	} else {
		return errors.New(errstr)
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: ignoressl},
	}
	baseurl, ok := (*(*c).monitor).monitorConfig["beaker_url"]
	if !ok {
		return errors.New(errstr)
	}

	device, ok := (*(*c).monitor).monitorConfig["beaker_device"]
	if !ok {
		return errors.New(errstr)
	}

	debugPrint(log.Printf, levelDebug, "get login page")
	loginURL := "https://" + baseurl + "/login"

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: tr}

	req, err := http.NewRequest("GET", loginURL, nil)
	if err != nil {
		debugPrint(log.Printf, levelError, err.Error())
		return errors.New("can't create request url\r\n")
	}
	req.Header.Set("User-Agent", "Provisioner")
	req.SetBasicAuth(username, password)

	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return err
	}
	debugPrint(log.Printf, levelDebug, "REQUEST:\n%s", string(reqDump))

	resp, err := client.Do(req)
	if err != nil {
		debugPrint(log.Printf, levelError, err.Error())
		return errors.New("can't get login page\r\n")
	}
	defer resp.Body.Close()

	respDump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}

	debugPrint(log.Printf, levelDebug, "RESPONSE:\n%s", string(respDump))

	if resp.StatusCode != http.StatusFound {
		return err
	}

	cookies := resp.Cookies()
	for _, cookie := range cookies {
		debugPrint(log.Printf, levelDebug, "Cookie: %s=%s\n", cookie.Name, cookie.Value)
	}
	cookie := resp.Cookies()[0]

	postURL := "https://" + baseurl + "/systems/" + device + "/commands/"

	postData := map[string]interface{}{
		"action": strings.ToLower(state),
	}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		return err
	}

	postReq, err := http.NewRequest("POST", postURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	postReq.AddCookie(cookie)

	postReq.SetBasicAuth(username, password)

	postReq.Header.Set("Content-Type", "application/json")

	reqDump, err = httputil.DumpRequestOut(postReq, true)
	if err != nil {
		return err
	}
	debugPrint(log.Printf, levelDebug, "REQUEST:\n%s", string(reqDump))

	postResp, err := client.Do(postReq)
	if err != nil {
		return err
	}
	defer postResp.Body.Close()

	body, err := ioutil.ReadAll(postResp.Body)
	if err != nil {
		return err
	}
	debugPrint(log.Printf, levelDebug, "POST body: %s", body)
	return nil
}
