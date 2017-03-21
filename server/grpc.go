package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/pims/fw/greeter"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"time"
)

const (
	port = ":50051"
)

var (
	// alpnProtoStr are the specified application level protocols for gRPC.
	alpnProtoStr = []string{"h2"}
)

type helloTlsCreds struct {
	// TLS configuration
	config *tls.Config
}

func (c helloTlsCreds) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "tls",
		SecurityVersion:  "1.2",
		ServerName:       c.config.ServerName,
	}
}

func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	if cfg == nil {
		return &tls.Config{}
	}

	return cfg.Clone()
}

func (c *helloTlsCreds) ClientHandshake(ctx context.Context, addr string, rawConn net.Conn) (_ net.Conn, _ credentials.AuthInfo, err error) {
	// use local cfg to avoid clobbering ServerName if using multiple endpoints
	cfg := cloneTLSConfig(c.config)
	if cfg.ServerName == "" {
		colonPos := strings.LastIndex(addr, ":")
		if colonPos == -1 {
			colonPos = len(addr)
		}
		cfg.ServerName = addr[:colonPos]
	}
	conn := tls.Client(rawConn, cfg)

	errChannel := make(chan error, 1)
	go func() {
		log.Println(time.Now())
		errChannel <- conn.Handshake()
	}()
	select {
	case err := <-errChannel:
		if err != nil {
			return nil, nil, err
		}
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
	return conn, credentials.TLSInfo{conn.ConnectionState()}, nil
}

func (c *helloTlsCreds) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {

	conn := tls.Server(rawConn, c.config)

	if err := conn.Handshake(); err != nil {
		return nil, nil, err
	}
	for _, peer := range conn.ConnectionState().PeerCertificates {
		fmt.Printf("%s %s\n", "-->", peer.Subject.CommonName)
	}
	return conn, credentials.TLSInfo{conn.ConnectionState()}, nil
}

func (c *helloTlsCreds) Clone() credentials.TransportCredentials {
	return NewTLS(c.config)
}

func NewTLS(c *tls.Config) credentials.TransportCredentials {
	tc := &helloTlsCreds{cloneTLSConfig(c)}
	tc.config.NextProtos = alpnProtoStr
	return tc
}

func (c *helloTlsCreds) OverrideServerName(serverNameOverride string) error {
	c.config.ServerName = serverNameOverride
	return nil
}

type server struct{}

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

	serverOption := grpc.Creds(NewTLS(tlsConfig))
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
