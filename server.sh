openssl genrsa -out server.key 2048
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=CA/L=San Francisco/O=Hello/OU=Pims/CN=sati.localhost"
# self-signed
openssl x509 -req -days 9999 -in server.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out server.crt