openssl genrsa -out ca.key 2048
openssl req -new -x509 -days 9999 -key ca.key -out ca.crt