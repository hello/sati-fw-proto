package main
import (
	"testing"
	"fmt"
)
func TestSyslog(t *testing.T) {
	digestPrinter := func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			digest := parseLog(logParts)
			fmt.Println(digest.Dumps())
		}
	}
	serverLoop(digestPrinter)
}
