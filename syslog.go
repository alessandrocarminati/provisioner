package main

import (
	"gopkg.in/mcuadros/go-syslog.v2"
	"log"
	"os"
	)
func syslog_service(fn string, port string) {
	debugPrint(log.Printf, levelWarning, "Starting syslog service on port %s -> %s ", port, fn)
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
	server.SetHandler(handler)
	server.ListenUDP("0.0.0.0:"+port)
	server.Boot()

	file, err := os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		debugPrint(log.Printf, levelError, err.Error())
	}
	defer file.Close()
	logger := log.New(file, "", log.LstdFlags)

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			debugPrint(log.Printf, levelDebug, "Received: %s", logParts["content"])
			logger.Printf("%s %d %d %s %s %s", logParts["timestamp"], logParts["severity"], logParts["priority"], logParts["hostname"], logParts["client"], logParts["content"])
		}
	}(channel)

	server.Wait()
}
