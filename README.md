## Go

```
$ brew install go
$ go version
go version go1.8 darwin/amd64

```
## Dependencies

See [http://www.grpc.io/docs/quickstart/go.html](http://www.grpc.io/docs/quickstart/go.html)

You might have to run some combination of `go get .`

## Client

```
go run client/grpc.go sati.localhost sati-pi

argv[1] = server address
argv[2] = folder where custom ssl cert is located

```

To add it to your hosts file:

```
echo "127.0.0.1    sati.locahost" | sudo tee -a /etc/hosts
```


## Server

```
go run server/grpc.go

# this loads server.crt and server.key from the current dir
```


## RaspberryPi

```
cat compile.sh

GOOS=linux GOARCH=arm GOARM=6 go build -o grpc-client-arm client/grpc.go


./compile.sh
```



## Keys

```
./ca.sh --> generates ca.crt & ca.key
./client.sh YOUR_DEVICE_NAME --> generates YOUR_DEVICE_NAME/YOUR_DEVICE_NAME.{crt|key} signed by CA
./server.sh --> generates server.{crt|key} for sati.locahost
```


## Protobuf

```
mkdir -p greeter
protoc greeter.proto --go_out=plugins=grpc:greeter
```
