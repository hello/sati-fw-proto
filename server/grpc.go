package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/hello/sati-fw-proto/greeter"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"
	"time"
)

const (
	port = ":50051"
)

var (
	// alpnProtoStr are the specified application level protocols for gRPC.
	alpnProtoStr = []string{"h2"}
)

type HelloCertStore interface {
	Exists(id string) (bool, error)
}

type InMemoryHelloCertStore struct {
	sync.Mutex
	m map[string]bool
}

func (s *InMemoryHelloCertStore) Exists(id string) (bool, error) {
	s.Lock()
	defer s.Unlock()
	_, found := s.m[id]
	return found, nil
}

func NewHelloTransportCredentialsChecker(c *tls.Config) credentials.TransportCredentials {
	m := make(map[string]bool)
	m["sati-pii"] = true
	return &HelloTransportCredentialsChecker{
		TransportCredentials: credentials.NewTLS(c),
		store:                &InMemoryHelloCertStore{m: m},
	}
}

type HelloTransportCredentialsChecker struct {
	credentials.TransportCredentials
	store HelloCertStore
}

func (c *HelloTransportCredentialsChecker) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	conn, authInfo, err := c.TransportCredentials.ServerHandshake(rawConn)
	if err != nil {
		log.Println("original handshake failed")
		return nil, nil, err

	}
	tlsInfo := authInfo.(credentials.TLSInfo)
	name := tlsInfo.State.PeerCertificates[0].Subject.CommonName
	found, err := c.store.Exists(name)
	if !found {
		conn.Close()
		return conn, authInfo, grpc.Errorf(codes.Unauthenticated, fmt.Sprintf("cert not found: %s", name))
	}

	fmt.Printf("%s\n", name)
	return conn, authInfo, err
}

type server struct{}

func (s *server) EmptyCall(ctx context.Context, in *greeter.Empty) (*greeter.Empty, error) {
	if md, ok := metadata.FromContext(ctx); ok {
		// For testing purpose, returns an error if there is attached metadata other than
		// the user agent set by the client application.
		if _, ok := md["user-agent"]; !ok {
			return nil, grpc.Errorf(codes.DataLoss, "missing expected user-agent")
		}
		var str []string
		for _, entry := range md["user-agent"] {
			str = append(str, "ua", entry)
		}
		grpc.SendHeader(ctx, metadata.Pairs(str...))
	}
	return new(greeter.Empty), nil
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *greeter.HelloRequest) (*greeter.HelloReply, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("invalid peer")
	}
	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	v := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	fmt.Printf("%v - %v\n", peer.Addr.String(), v)

	return &greeter.HelloReply{Message: "Hello " + v}, nil
}

func (s *server) Periodic(stream greeter.Greeter_PeriodicServer) error {
	peer, ok := peer.FromContext(stream.Context())
	if !ok {
		return errors.New("invalid peer cert")
	}
	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	v := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	fmt.Printf("%v - %v\n", peer.Addr.String(), v)

	go func() {
		for {
			in, err := stream.Recv()

			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("%v", err)
			}
			fmt.Println("server:", in.Name)
		}
	}()

	for {
		// fmt.Printf("Message: %s %s\n", n.Channel, n.Data)
		rep := &greeter.HelloReply{Message: fmt.Sprintf("%s: %s", v, time.Now())}
		if err := stream.Send(rep); err != nil {
			log.Fatal(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func serverFunc() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		log.Fatal(err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile("ca.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
	}

	serverOption := grpc.Creds(NewHelloTransportCredentialsChecker(tlsConfig))
	s := grpc.NewServer(serverOption)
	greeter.RegisterGreeterServer(s, &server{})
	log.Println("Serving...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	serverFunc()
}
