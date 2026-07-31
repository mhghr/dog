#!/usr/bin/env bash
# Generates a self-signed cert for local otel-ingest TLS testing.
set -euo pipefail
mkdir -p deployments/certs
openssl req -x509 -newkey rsa:2048 -nodes \
  -keyout deployments/certs/server.key \
  -out deployments/certs/server.crt \
  -days 365 \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
echo "certs written to deployments/certs/"
