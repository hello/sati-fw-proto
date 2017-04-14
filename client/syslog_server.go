package main

import (
	"errors"
	"fmt"
	"testing"
	"time"
	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
	"github.com/hello/sati-fw-proto/greeter"
)

var parserError = errors.New("Unable to parse log")

type logDigest struct {
	severity int
	app_name string
	message  string
	ts	 	 time.Time
}
func (d logDigest)Dumps() string {
	return fmt.Sprintf("%d (%d)%s:%s", d.ts.Unix(), d.severity, d.app_name, d.message)
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
func getDefaultTime(p format.LogParts, key string, defaultVal time.Time) time.Time {
	if val, ok := p[key]; ok {
		if ret, ok := val.(time.Time); ok {
			return ret
		}
	}
	return defaultVal
}
func parseLog(part format.LogParts) (ret logDigest) {
	ret.severity = getDefaultInt(part, "severity", 9)
	ret.app_name = getDefaultString(part, "app_name", "")
	ret.message  = getDefaultString(part, "message", "")
	ret.ts 		 = getDefaultTime(part, "timestamp", time.Now())
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
func SyslogServerLoop(outboundChannel chan<- greeter.LogEntry) {
	digest := func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			fmt.Println("Got something")
			digest := parseLog(logParts)
			outboundChannel<- greeter.LogEntry{
				Severity : int32(digest.severity),
				AppName  : digest.app_name,
				Text     : digest.message,
			}
		}
	}
	serverLoop(digest)
}
func TestSyslog(t *testing.T) {
/*
 *func main() {
 */
	digestPrinter := func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			digest := parseLog(logParts)
			fmt.Println(digest.Dumps())
		}
	}
	serverLoop(digestPrinter)
}
