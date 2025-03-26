package certs

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"
)

// Provider defines the interface for certificate providers
type Provider interface {
	GetCertificate() *tls.Certificate
	Start(ctx context.Context) error
	Stop()
}

// FSProvider implements certificate loading from filesystem
type FSProvider struct {
	domain     string
	certPath   string
	keyPath    string
	reloadFreq time.Duration

	mu       sync.RWMutex
	cert     tls.Certificate
	stopChan chan struct{}
	logger   Logger
}

// Logger interface allows for flexible logging implementation
type Logger interface {
	Errorf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}

// NewFSProvider creates a new filesystem-based certificate provider
func NewFSProvider(domain string, certPath, keyPath string, logger Logger) *FSProvider {
	return &FSProvider{
		domain:     domain,
		certPath:   certPath,
		keyPath:    keyPath,
		reloadFreq: 24 * time.Hour,
		stopChan:   make(chan struct{}),
		logger:     logger,
	}
}

// loadCertificates loads certificates from filesystem
func (p *FSProvider) loadCertificates() error {
	cert, err := tls.LoadX509KeyPair(p.certPath, p.keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificates: %w", err)
	}

	p.mu.Lock()
	p.cert = cert
	p.mu.Unlock()

	return nil
}

// GetCertificate returns the current certificate
func (p *FSProvider) GetCertificate() *tls.Certificate {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return &p.cert
}

// Start begins the certificate reload loop
func (p *FSProvider) Start(ctx context.Context) error {
	// Initial load
	if err := p.loadCertificates(); err != nil {
		return err
	}
	p.logger.Infof("Initial certificates loaded for domain: %s", p.domain)

	// Start reload loop
	go func() {
		ticker := time.NewTicker(p.reloadFreq)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-p.stopChan:
				return
			case <-ticker.C:
				if err := p.loadCertificates(); err != nil {
					p.logger.Errorf("Failed to reload certificates: %v", err)
					continue
				}
				p.logger.Infof("Certificates successfully reloaded for domain: %s", p.domain)
			}
		}
	}()

	return nil
}

// Stop halts the certificate reload loop
func (p *FSProvider) Stop() {
	close(p.stopChan)
}
