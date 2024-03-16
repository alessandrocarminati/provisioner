package main

import (
	"gopkg.in/mcuadros/go-syslog.v2"
	"log"
	"os"
	)
func syslog_service(fn string, port string) {
	log.Printf("Starting syslog service on port %s -> %s ", port, fn)
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	server.ListenUDP("0.0.0.0:"+port)
	server.Boot()

	file, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	logger := log.New(file, "", log.LstdFlags)

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			logger.Println(logParts)
		}
	}(channel)

	server.Wait()
}
