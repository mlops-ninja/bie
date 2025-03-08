package biecy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"time"
)

func GenerateMinimalCA() ([]byte, *rsa.PrivateKey) {
	caKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "BieCA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	caBytes, _ := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes})
	return caPEM, caKey
}

func GenerateMinimalServerCert(caCert []byte, caKey *rsa.PrivateKey, domain string) ([]byte, []byte) {
	serverKey, _ := rsa.GenerateKey(rand.Reader, 1024)
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: domain},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	block, _ := pem.Decode(caCert)
	if block == nil {
		log.Fatal("Failed to decode PEM block")
	}
	caParsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatal("Failed to parse CA certificate:", err)
	}
	serverBytes, _ := x509.CreateCertificate(rand.Reader, serverTemplate, caParsed, &serverKey.PublicKey, caKey)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	return certPEM, keyPEM
}
