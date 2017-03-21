#!/bin/sh

NAME=$1
mkdir -p $NAME
openssl genrsa -out client.key 2048
openssl req -new -key client.key -out client.csr -subj "/C=US/ST=CA/L=San Francisco/O=Hello/OU=Pims/CN=$NAME"
cp client.key $NAME/$NAME.key
# self-signed
openssl x509 -req -days 9999 -in client.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out $NAME/$NAME.crt