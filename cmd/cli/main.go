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
	"strings"
	"time"

	"bie/pkg/biecy"
	"bie/pkg/biewire"
	"bie/pkg/osserver"

	"github.com/alecthomas/kong"
	"github.com/caarlos0/env/v11"

	tea "github.com/charmbracelet/bubbletea"
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

	dialer := net.Dialer{Timeout: 30 * time.Second}
	conn, err := dialer.Dial("tcp", cfg.ServerAddress)
	if err != nil {
		return fmt.Errorf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte{biewire.OpGet}); err != nil {
		return fmt.Errorf("Failed to send opcode: %v", err)
	}

	buffer := make([]byte, 256)
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("Failed to read from server: %v", err)
	}
	base32String := strings.TrimSpace(string(buffer[:n]))
	bieDomain := base32String + "." + cfg.Domain

	caCert, caKey := biecy.GenerateMinimalCA()
	certPEM, keyPEM := biecy.GenerateMinimalServerCert(caCert, caKey, bieDomain)

	// Creating TUI
	curlCmd := fmt.Sprintf(
		"curl -X POST -k -F 'file=@%s' --cacert <(echo '%s') https://%s:%d/file",
		targetFile,
		string(certPEM),
		bieDomain,
		cfg.Port,
	)
	p := tea.NewProgram(Model{FilePath: targetFile, Command: curlCmd, FileSize: 0, Uploaded: 0})

	// go func() {
	// 	// Starting TUI ins separate goroutine
	// 	if err := p.Start(); err != nil {
	// 		log.Fatalf("Error running TUI: %v", err)
	// 	}
	// }()
	log.Printf("Run the following command to upload the file:\n%s", curlCmd)

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("Failed to load TLS certificate: %v", err)
	}

	tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{cert}})
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("TLS handshake failed: %v", err)
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
		fmt.Println("File saved. Exiting.")
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "File %s successfully transfered\n", targetFile)
	})

	// Server
	server := osserver.NewOneShotServer(tlsConn, mux)
	if err := server.Serve(context.Background()); err != nil {
		return fmt.Errorf("Server error: %v", err)
	}

	// p.Send(struct{}{})
	fmt.Println("File uploaded. Exiting.", p)

	// if err := p.Start(); err != nil {
	// 	return fmt.Errorf("Error running TUI: %v", err)
	// }

	return nil
}

var CLI struct {
	Get GetCmd `cmd:"" help:"Get a file."`
}

func main() {
	ctx := kong.Parse(&CLI, kong.UsageOnError())
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
