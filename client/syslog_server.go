package main
import "gopkg.in/mcuadros/go-syslog.v2"
import "fmt"

func main() {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)
	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	listenErr := server.ListenUDP("0.0.0.0:514")
	err := server.Boot()
	if err != nil || listenErr != nil {
		fmt.Println("Error is", err)
		fmt.Println("Error is", listenErr)
	}

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			 for k,v := range logParts {
				 fmt.Println(k,":", v)
			 }
		}
	}(channel)
	server.Wait()
	fmt.Printf("Server Exit")
}


