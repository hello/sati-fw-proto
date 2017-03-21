package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/pims/fw/greeter"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"
)

func client(addr, crt, key string) {

	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		log.Println("failed: LoadX509KeyPair", crt, key)
		log.Fatal(err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile("ca.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	transportCreds := credentials.NewTLS(&tls.Config{
		ServerName:   addr,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	})

	dialOption := grpc.WithTransportCredentials(transportCreds)
	conn, err := grpc.Dial(addr+":50051", dialOption)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	ctx := context.Background()
	stream, err := greeter.NewGreeterClient(conn).Periodic(ctx)
	// c := greeter.NewGreeterClient(conn)

	// Contact the server and print out its response.
	sendTicker := time.NewTicker(100 * time.Millisecond)

	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				// TODO: handle error
				log.Fatalf("%v", err)
			}
			fmt.Println("client:", resp.Message)
		}
	}()

	for {
		select {
		case <-sendTicker.C:
			err3 := stream.Send(&greeter.HelloRequest{Name: fmt.Sprintf("name %d", time.Now().Unix())})
			if err3 != nil {
				log.Fatal(err3)
			}
		}
	}
}

func main() {

	addr := os.Args[1]
	name := os.Args[2]

	crt := fmt.Sprintf("%s/%s.crt", name, name)
	key := fmt.Sprintf("%s/%s.key", name, name)
	client(addr, crt, key)
}
