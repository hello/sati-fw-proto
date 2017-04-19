package main

import (
	"fmt"
	"log"
	"time"
	"github.com/hello/sati-fw-proto/greeter"
	zmq "github.com/pebble/zmq4"
)

func ExamplePub() {
	pub, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		log.Fatal("ZMG!!!")
		return
	}
	defer pub.Close()
	if err := pub.Bind("ipc:///tmp/hello"); err != nil {
		log.Fatal("Conn Error", err)
		return
	}
	tick := time.Tick(time.Millisecond * 1000)
	for {
		select {
		case <- tick:
			if _,err := pub.Send(fmt.Sprint("time %s", time.Now().String()), 0); err != nil {
				log.Fatal(err)
				break
			}
		}
	}

}
func ExampleSub(out chan<- *greeter.HelloRequest) {
	receiver, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatal("ZMG!!!")
		return
	}
	defer receiver.Close()
	if err := receiver.Connect("ipc:///tmp/hello"); err != nil {
		log.Fatal("Conn Error", err)
		return
	}
	if err := receiver.SetSubscribe("time"); err != nil {
		log.Fatal("Sub Error", err)
		return
	}
	for {
		msg, err := receiver.Recv(0)
		if err != nil {
			break
		} else {
			fmt.Println(msg)
		}
	}

}

func main() {
	c := make(chan *greeter.HelloRequest, 100)
	go ExampleSub(c)
	ExamplePub()
}
