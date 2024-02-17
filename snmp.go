package main

import (
	"time"

	"github.com/gosnmp/gosnmp"
)

func snmpSetv2(oid string, value interface{}, target string, community string) error {
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
func snmpSetv3unsec(oid string, value interface{}, target string, username string) error {
	params := &gosnmp.GoSNMP{
		Target:          target,
		Port:            161,
		Version:         gosnmp.Version3,
		Timeout:         time.Duration(2) * time.Second,
		Retries:         3,
		MaxOids:         1,
		Transport:       "udp",
		SecurityModel:   gosnmp.UserSecurityModel,
		MsgFlags:        gosnmp.NoAuthNoPriv,
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
