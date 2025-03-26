package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"bie/pkg/biecy"
	"bie/pkg/biewire"
	"bie/pkg/osserver"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v11"

	"github.com/xtaci/smux"
)

type Config struct {
	ServerAddress string `env:"BIE_SERVER" envDefault:"bie.mlops.ninja:80"`
	Port          int    `env:"BIE_PORT" envDefault:"443"`
	Domain        string `env:"BIE_DOMAIN" envDefault:"bie.mlops.ninja"`
}

type GetCmd struct {
	NewFilePath string `arg:"" name:"new-file-path" help:"Path to save the file to." type:"path"`
}

func (c *GetCmd) Run() error {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	targetFile := c.NewFilePath

	// 1. Connect to relay with TLS
	tlsConn, err := tls.DialWithDialer(
		&net.Dialer{
			Timeout: 30 * time.Second,
		},
		"tcp",
		cfg.ServerAddress,
		&tls.Config{
			ServerName: cfg.Domain, // Required for SNI and certificate validation
		},
	)
	if err != nil {
		return fmt.Errorf("TLS connection failed: %v", err)
	}

	// 2. Create smux session
	session, err := smux.Client(tlsConn, nil)
	if err != nil {
		return fmt.Errorf("Failed to create smux session: %v", err)
	}

	// 3. Create auth stream
	authStream, err := session.OpenStream()
	if err != nil {
		return fmt.Errorf("Failed to open auth stream: %v", err)
	}
	defer authStream.Close()

	// 2. Send auth request
	if err := sendAuthRequest(authStream, "", "get"); err != nil {
		session.Close()
		return fmt.Errorf("Failed to send request: %v", err)
	}

	// 3. Read token from server
	var clientResponse biewire.ClientResponse
	if biewire.ReceiveJSON(authStream, &clientResponse); err != nil {
		session.Close()
		return fmt.Errorf("Failed to read response: %v", err)
	}

	bieDomain := clientResponse.Token + "." + cfg.Domain

	// 5. Generate our own certificate for the server role
	caCert, caKey := biecy.GenerateMinimalCA()
	certPEM, keyPEM := biecy.GenerateMinimalServerCert(caCert, caKey, bieDomain)

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		session.Close()
		return fmt.Errorf("Failed to load certificate: %v", err)
	}

	// Create TLS config for our server
	serverTLSConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Creating TUI
	curlCmd := fmt.Sprintf(
		"curl -X POST -k -F 'file=@%s' --cacert <(echo '%s') https://%s:%d/file",
		targetFile,
		string(certPEM),
		bieDomain,
		cfg.Port,
	)
	fmt.Println(curlCmd)
	// p := tea.NewProgram(Model{FilePath: targetFile, Command: curlCmd, FileSize: 0, Uploaded: 0} /*tea.WithAltScreen()*/)

	// go func() {
	// 	if _, err := p.Run(); err != nil {
	// 		log.Fatalf("Error running TUI: %v", err)
	// 	}
	// 	os.Exit(0)
	// }()

	// 6. Accepting incoming stream from relay
	serverStream, err := session.AcceptStream()
	if err != nil {
		session.Close()
		return fmt.Errorf("Failed to accept stream: %v", err)
	}

	serverTLSConn := tls.Server(serverStream, serverTLSConfig)
	if err := serverTLSConn.Handshake(); err != nil {
		session.Close()
		return fmt.Errorf("Server TLS handshake failed: %v", err)
	}

	// Mux for oneshot server
	mux := http.NewServeMux()
	mux.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to read file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		out, err := os.Create(targetFile)
		if err != nil {
			http.Error(w, "Failed to create file", http.StatusInternalServerError)
			return
		}
		defer out.Close()
		io.Copy(out, file)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "File %s successfully transfered\n", targetFile)
	})

	// Server with TLS
	server := osserver.NewOneShotServer(serverTLSConn, mux)
	if err := server.Serve(context.Background()); err != nil {
		return fmt.Errorf("Server error: %v", err)
	}

	// p.Quit()
	return nil
}

// Send request to server
func sendAuthRequest(conn io.Writer, authToken string, intention string) error {
	// Create request
	req := biewire.ClientRequest{
		Intention: intention,
		AuthToken: authToken,
	}

	return biewire.SendJSON(conn, req)
}

var CLI struct {
	Get GetCmd `cmd:"" help:"Get a file."`
}

func main() {
	ctx := kong.Parse(&CLI, kong.UsageOnError())
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
