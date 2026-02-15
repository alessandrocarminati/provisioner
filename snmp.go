package main

import (
	"errors"
	"github.com/gosnmp/gosnmp"
	"strconv"
	"time"
)

func (c *CmdCtx) snmpSwitch(state string) error {
	var val int
	var err error

	errstr := "SNMP Config error"
	oid, ok := (*(*c).monitor).monitorConfig["snmp_pdu_ctrl_oid"]
	if !ok {
		return errors.New(errstr)
	}
	host, ok := (*(*c).monitor).monitorConfig["snmp_pdu_ctrl_host"]
	if !ok {
		return errors.New(errstr)
	}
	user, ok := (*(*c).monitor).monitorConfig["snmp_pdu_ctrl_user"]
	if !ok {
		return errors.New(errstr)
	}
	onValue, ok := (*(*c).monitor).monitorConfig["snmp_pdu_ctrl_on_val"]
	if !ok {
		return errors.New(errstr)
	}
	offValue, ok := (*(*c).monitor).monitorConfig["snmp_pdu_ctrl_off_val"]
	if !ok {
		return errors.New(errstr)
	}
	if (state == "ON") || (state == "OFF") {
		if state == "ON" {
			val, err = strconv.Atoi(onValue)
		} else {
			val, err = strconv.Atoi(offValue)
		}
		if err != nil {
			return errors.New(errstr)
		}
		err := c.snmpSetv3unsec(oid, val, host, user)
		return err
	}
	return errors.New("Unknown state")
}

func (c *CmdCtx) snmpSetv2(oid string, value interface{}, target string, community string) error {
	params := &gosnmp.GoSNMP{
		Target:    target,
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(2) * time.Second,
		Retries:   3,
	}
	err := params.Connect()
	if err != nil {
		return err
	}
	defer params.Conn.Close()

	pdu := gosnmp.SnmpPDU{
		Name:  oid,
		Type:  gosnmp.OctetString,
		Value: value,
	}

	_, err = params.Set([]gosnmp.SnmpPDU{pdu})
	if err != nil {
		return err
	}

	return nil
}
func (c *CmdCtx) snmpSetv3unsec(oid string, value interface{}, target string, username string) error {
	params := &gosnmp.GoSNMP{
		Target:        target,
		Port:          161,
		Version:       gosnmp.Version3,
		Timeout:       time.Duration(2) * time.Second,
		Retries:       3,
		MaxOids:       1,
		Transport:     "udp",
		SecurityModel: gosnmp.UserSecurityModel,
		MsgFlags:      gosnmp.NoAuthNoPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName: username,
		},
	}

	err := params.Connect()
	if err != nil {
		return err
	}
	defer params.Conn.Close()

	pdu := gosnmp.SnmpPDU{
		Name:  oid,
		Type:  gosnmp.Integer,
		Value: value,
	}

	_, err = params.Set([]gosnmp.SnmpPDU{pdu})
	if err != nil {
		return err
	}

	return nil
}
