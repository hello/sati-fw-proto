package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/hello/sati-fw-proto/greeter"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

/*
 *SyslogOutbound   chan HelloService.LogEntry
 *PeriodicOutbound chan HelloService.HelloRequest
 *PeriodicInbound  chan HelloService.HelloReply
 */
func getDialOptions(addr, crt, key string) ([]grpc.DialOption, error) {
	cert, err := tls.LoadX509KeyPair(crt, key)
	if err != nil {
		log.Fatal("Failed: LoadX509KeyPair", crt, key)
		return nil, err
	}

	caCert, err := ioutil.ReadFile("ca.crt")
	if err != nil {
		log.Fatal("Failed: LoadCA", crt, key)
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	transportCreds := credentials.NewTLS(&tls.Config{
		ServerName:   addr,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	})

	backOffConfig := grpc.BackoffConfig{
		MaxDelay: 10 * time.Second,
	}

	return []grpc.DialOption{
		// grpc.WithTimeout(500 * time.Millisecond),
		grpc.WithBackoffConfig(backOffConfig),
		grpc.WithTransportCredentials(transportCreds),
		grpc.WithUserAgent("grpc-go-client"),
	}, nil
}
func client(addr, crt, key string) {
	dialOptions, err := getDialOptions(addr, crt, key)
	if err != nil {
		log.Fatal(err)
		return
	}
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, addr+":50051", dialOptions...)

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()

	c := greeter.NewGreeterClient(conn)

	// Important to attempt the first call when starting to start the tls negotiation check
	if _, err := c.EmptyCall(ctx, &greeter.Empty{}, grpc.FailFast(true)); err != nil {
		log.Printf("%v", err)
		return
	}

	stream, err := greeter.NewGreeterClient(conn).Periodic(ctx)
	logStream, err := greeter.NewGreeterClient(conn).Syslog(ctx)
	// Contact the server and print out its response.
	sendTicker := time.NewTicker(500 * time.Millisecond)
	logChan := make(chan greeter.LogEntry, 100)

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

	go SyslogServerLoop(logChan)
	for {
		select {
		case <-stream.Context().Done():
			closeErr := stream.CloseSend()
			if closeErr != nil {
				log.Fatal(closeErr)
			}
		case <-sendTicker.C:
			err3 := stream.Send(&greeter.HelloRequest{Name: fmt.Sprintf("name %d", time.Now().Unix())})
			if err3 != nil {
				log.Fatal("err3", err3)
			}
		case l := <-logChan:
			err4 := logStream.Send(&l)
			if err4 != nil {
				log.Fatal("err4", err4)
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
