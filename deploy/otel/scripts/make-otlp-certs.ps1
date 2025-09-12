# Generate OTLP TLS certs for Local Dev (PowerShell)
# - Creates a local CA (ca.key/ca.pem)
# - Creates a server cert/key for the collector with SAN=DNS:otel-collector
# - Outputs files into deploy/otel/tls and deploy/otel/ca
#
# Usage (from repo root in PowerShell):
#   pwsh .\deploy\otel\scripts\make-otlp-certs.ps1
#
# Requires: OpenSSL in PATH.

param()

$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$tlsDir = Join-Path $root '..\tls'
$caDir  = Join-Path $root '..\ca'

New-Item -ItemType Directory -Force -Path $tlsDir | Out-Null
New-Item -ItemType Directory -Force -Path $caDir  | Out-Null

$caKey = Join-Path $tlsDir 'ca.key'
$caPem = Join-Path $caDir  'ca.pem'
$serverKey = Join-Path $tlsDir 'server.key'
$serverCsr = Join-Path $tlsDir 'server.csr'
$serverCrt = Join-Path $tlsDir 'server.crt'
$confFile  = Join-Path $tlsDir 'openssl-san.cnf'

# Create OpenSSL config with SAN=otel-collector
@"
[ req ]
default_bits       = 2048
distinguished_name = req_distinguished_name
req_extensions     = req_ext
prompt             = no

[ req_distinguished_name ]
C  = US
ST = State
L  = Local
O  = PromptShield
OU = Dev
CN = otel-collector

[ req_ext ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = otel-collector
"@ | Out-File -Encoding ascii $confFile

Write-Host 'Generating CA key and certificate...'
if (-not (Test-Path $caKey)) {
  openssl genrsa -out $caKey 4096 | Out-Null
}
openssl req -x509 -new -nodes -key $caKey -sha256 -days 3650 -out $caPem -subj "/C=US/ST=State/L=Local/O=PromptShield/OU=Dev CA/CN=PromptShield Dev CA" | Out-Null

Write-Host 'Generating server key and CSR...'
openssl genrsa -out $serverKey 2048 | Out-Null
openssl req -new -key $serverKey -out $serverCsr -config $confFile | Out-Null

Write-Host 'Signing server certificate with CA...'
openssl x509 -req -in $serverCsr -CA $caPem -CAkey $caKey -CAcreateserial -out $serverCrt -days 825 -sha256 -extensions req_ext -extfile $confFile | Out-Null

Write-Host 'Done.'
Write-Host "Collector TLS files:"
Write-Host "  $serverCrt"
Write-Host "  $serverKey"
Write-Host "App CA file:"
Write-Host "  $caPem"

