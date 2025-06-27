#!/bin/bash
# This script generates a self-signed CA and a server certificate for the webhook service
# Usage: ./generate-certs.sh <service-name> <namespace>

set -e

SERVICE_NAME=${1:-webhook-server}
NAMESPACE=${2:-webhook-demo}
TMPDIR=$(mktemp -d)

cat > $TMPDIR/openssl.cnf <<EOF
[req]
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_ca ]
subjectKeyIdentifier=hash
authorityKeyIdentifier=keyid:always,issuer:always
basicConstraints = critical,CA:true
keyUsage = critical, digitalSignature, cRLSign, keyCertSign
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${SERVICE_NAME}
DNS.2 = ${SERVICE_NAME}.${NAMESPACE}
DNS.3 = ${SERVICE_NAME}.${NAMESPACE}.svc
EOF

# Generate CA key and cert
openssl genrsa -out $TMPDIR/ca.key 2048
openssl req -x509 -new -nodes -key $TMPDIR/ca.key -subj "/CN=webhook-ca" -days 3650 -out $TMPDIR/ca.crt -config $TMPDIR/openssl.cnf -extensions v3_ca

# Generate server key and CSR
openssl genrsa -out $TMPDIR/server.key 2048
openssl req -new -key $TMPDIR/server.key -subj "/CN=${SERVICE_NAME}.${NAMESPACE}.svc" -out $TMPDIR/server.csr -config $TMPDIR/openssl.cnf -reqexts v3_req

# Sign server cert with CA
openssl x509 -req -in $TMPDIR/server.csr -CA $TMPDIR/ca.crt -CAkey $TMPDIR/ca.key -CAcreateserial -out $TMPDIR/server.crt -days 3650 -extensions v3_req -extfile $TMPDIR/openssl.cnf

# Output base64 for Kubernetes manifests
cat $TMPDIR/server.crt | base64 | tr -d '\n' > tls.crt.b64
cat $TMPDIR/server.key | base64 | tr -d '\n' > tls.key.b64
cat $TMPDIR/ca.crt | base64 | tr -d '\n' > ca.crt.b64

cp $TMPDIR/server.crt ./tls.crt
cp $TMPDIR/server.key ./tls.key
cp $TMPDIR/ca.crt ./ca.crt

rm -rf $TMPDIR

echo "Certificates generated: tls.crt, tls.key, ca.crt, tls.crt.b64, tls.key.b64, ca.crt.b64"
echo "Use the contents of tls.crt.b64 and tls.key.b64 in your tls-secret.yaml, and ca.crt.b64 in your webhook.yaml CA_BUNDLE."
