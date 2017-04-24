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

type HelloService struct {
	addr             string
	crt              string
	key              string
	SyslogOutbound   chan *greeter.LogEntry
	PeriodicOutbound chan *greeter.HelloRequest
	PeriodicInbound  chan *greeter.HelloReply
}

func NewHelloService(addr, crt, key string) *HelloService {
	return &HelloService{
		addr:             addr,
		crt:              crt,
		key:              key,
		SyslogOutbound:   make(chan *greeter.LogEntry, 100),
		PeriodicOutbound: make(chan *greeter.HelloRequest, 100),
		PeriodicInbound:  make(chan *greeter.HelloReply, 100),
	}
}
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
func (srv *HelloService) receivePeriodic(stream greeter.Greeter_PeriodicClient) error {
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			// TODO: handle error
			log.Fatalf("%v", err)
			return err
		}
		fmt.Println("client:", resp.Message)
		srv.PeriodicInbound <- resp
	}
	return nil
}
func (srv *HelloService) Close() {
	close(srv.SyslogOutbound)
	close(srv.PeriodicOutbound)
	close(srv.PeriodicInbound)
}
func (srv *HelloService) ClientLoop() error {
	dialOptions, err := getDialOptions(srv.addr, srv.crt, srv.key)
	if err != nil {
		log.Fatal(err)
		return err
	}
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, srv.addr+":50051", dialOptions...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return err
	}
	defer conn.Close()

	// Important to attempt the first call when starting to start the tls negotiation check
	c := greeter.NewGreeterClient(conn)
	if _, err := c.EmptyCall(ctx, &greeter.Empty{}, grpc.FailFast(true)); err != nil {
		log.Fatal(err)
		return err
	}

	//Make streams
	periodicStream, err := greeter.NewGreeterClient(conn).Periodic(ctx)
	if err != nil {
		return err
	} else {
		defer periodicStream.CloseSend()
	}

	logStream, err := greeter.NewGreeterClient(conn).Syslog(ctx)
	if err != nil {
		return err
	} else {
		defer logStream.CloseSend()
	}

	//inbound loops
	go srv.receivePeriodic(periodicStream)
	//outbound loop
	var e error = nil
	for {
		select {
		case <-logStream.Context().Done():
			break
		case <-periodicStream.Context().Done():
			break
		case l := <-srv.SyslogOutbound:
			e = logStream.Send(l)
			if e != nil {
				break
			}
		case l := <-srv.PeriodicOutbound:
			e = periodicStream.Send(l)
			if e != nil {
				break
			}
		}

	}
	if e != nil {
		log.Fatal("Outbound error", e)
	}
	return e
}

func main() {

	addr := os.Args[1]
	name := os.Args[2]

	crt := fmt.Sprintf("%s/%s.crt", name, name)
	key := fmt.Sprintf("%s/%s.key", name, name)
	c := NewHelloService(addr, crt, key)
	go SyslogServerLoop(c.SyslogOutbound)
	go func(c chan *greeter.HelloRequest) {
		tick := time.Tick(time.Millisecond * 500)
		for {
			select {
			case <-tick:
				c <- &greeter.HelloRequest{
					Name: time.Now().String(),
				}

			}
		}
	}(c.PeriodicOutbound)
	c.ClientLoop()
	c.Close()
}
