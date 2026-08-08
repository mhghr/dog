package agents

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

type CertAuthority struct {
	CACert *x509.Certificate
	CAKey  crypto.Signer
}

func NewCertAuthority(caCertPEM, caKeyPEM []byte) (*CertAuthority, error) {
	certBlock, _ := pem.Decode(caCertPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	keyBlock, _ := pem.Decode(caKeyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode CA key PEM")
	}

	var caKey crypto.Signer
	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		parsedKey, parseErr := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse RSA CA key: %w", parseErr)
		}
		caKey = parsedKey
	case "PRIVATE KEY":
		key, parseErr := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse CA key: %w", parseErr)
		}
		var ok bool
		caKey, ok = key.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("CA key does not implement crypto.Signer")
		}
	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyBlock.Type)
	}

	return &CertAuthority{CACert: caCert, CAKey: caKey}, nil
}

func NewSelfSignedCA() (*CertAuthority, []byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate CA key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate serial: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "Monitoring Platform Probe Agent CA",
		},
		NotBefore:             now,
		NotAfter:              now.Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	caCert, _ := x509.ParseCertificate(certDER)
	ca := &CertAuthority{CACert: caCert, CAKey: priv}

	return ca, certPEM, keyPEM, nil
}

func (ca *CertAuthority) Fingerprint() string {
	hash := sha256.Sum256(ca.CACert.Raw)
	return hex.EncodeToString(hash[:])
}

func (ca *CertAuthority) IssueAgentCertificate(agentID, hostname string, publicKey crypto.PublicKey) (certPEM, serial string, err error) {
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial: %w", err)
	}
	serial = serialNumber.Text(16)

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("probe-agent-%s", agentID),
		},
		NotBefore:             now,
		NotAfter:              now.Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.CACert, publicKey, ca.CAKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM = string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	}))

	return certPEM, serial, nil
}
