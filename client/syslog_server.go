package main

import (
	"errors"
	"fmt"

	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
)

var parserError = errors.New("Unable to parse log")

type logDigest struct {
	severity int
	app_name string
	message  string
}
func (d logDigest)Dumps() string {
	return fmt.Sprintf("(%d)%s:%s", d.severity, d.app_name, d.message)
}

func getDefaultInt(p format.LogParts, key string, defaultVal int) int {
	if val, ok := p[key]; ok {
		if ret, ok := val.(int); ok {
			return ret
		}
	}
	return defaultVal
}
func getDefaultString(p format.LogParts, key string, defaultVal string) string {
	if val, ok := p[key]; ok {
		if ret, ok := val.(string); ok {
			return ret
		}
	}
	return defaultVal
}
func parseLog(part format.LogParts) (ret logDigest) {
	ret.severity = getDefaultInt(part, "severity", 9)
	ret.app_name = getDefaultString(part, "app_name", "")
	ret.message = getDefaultString(part, "message", "")
	return
}
func serverLoop(cb func(syslog.LogPartsChannel)) error {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	if err := server.ListenUDP("0.0.0.0:514"); err != nil {
		fmt.Println("Error: ", err)
		return err
	}
	if err := server.Boot(); err != nil {
		fmt.Println("Error: ", err)
		return err
	}
	go cb(channel)
	server.Wait()
	fmt.Printf("Server Exit")

	return nil
}
func digestPrinter(channel syslog.LogPartsChannel) {
	for logParts := range channel {
		digest := parseLog(logParts)
		fmt.Println(digest.Dumps())
	}
}
func main() {
	serverLoop(digestPrinter)
}
