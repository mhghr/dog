package transport

import "crypto/tls"

// TLSConfig returns the agent's hardened client TLS configuration.
//
// mTLS is not enabled yet; to switch the authentication layer to mutual TLS,
// set ClientCertificates to an agent certificate/key pair issued by Core and
// verify the server chain via RootCAs. No business logic changes are required
// for this swap — the credential layer is behind the Authenticator interface.
func TLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
		CurvePreferences: []tls.CurveID{
			tls.X25519,
			tls.CurveP384,
		},
	}
}
