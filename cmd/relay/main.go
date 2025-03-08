package main

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"bie/pkg/bielog"
	"bie/pkg/biewire"

	"github.com/caarlos0/env/v11"
)

// Relay server settings
const (
	relayPort = ":443"
	tokenSize = 32 // 32 random bytes (256-bit security)
)

type Config struct {
	ServerAddress string `env:"BIE_SERVER" envDefault:"bie.mlops.ninja:80"`
	SenderPort    int    `env:"BIE_SENDER_PORT" envDefault:"443"`
	ReceiverPort  int    `env:"BIE_RECEIVER_PORT" envDefault:"5443"`
	Domain        string `env:"BIE_DOMAIN"`
	ShardID       string `env:"SHARD_ID" envDefault:"01"`
	// Logger
	LogType  string `env:"BIE_LOG_TYPE" envDefault:"text"`
	LogLevel string `env:"BIE_LOG_LEVEL" envDefault:"info"`
}

// Store active connections (Token → Connection)
var connectionStore = struct {
	sync.RWMutex
	connections map[string]net.Conn
}{connections: make(map[string]net.Conn)}

// Generates a secure random `XID` token
func generateSecureToken() string {
	randomBytes := make([]byte, tokenSize)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatal("Failed to generate secure token:", err)
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes) // Base32 encoding
}

// Handles receiver registration
func registerReceiver(conn net.Conn, cfg Config) {
	defer conn.Close()

	// First read opcode to ensure it's a receiver
	buffer := make([]byte, 1)
	_, err := conn.Read(buffer)
	if err != nil {
		log.Println("Error reading receiver opcode:", err)
		return
	}
	if buffer[0] != biewire.OpGet {
		log.Println("Invalid receiver opcode")
	}

	// Generate `SHARD-ID-XID`
	shardID := cfg.ShardID
	xid := generateSecureToken()
	token := strings.ToLower(fmt.Sprintf("%s-%s", shardID, xid))

	// Send token to receiver
	_, err = conn.Write([]byte(token + "\n"))
	if err != nil {
		log.Println("Error sending token:", err)
		return
	}

	// Store receiver connection
	connectionStore.Lock()
	connectionStore.connections[token] = conn
	connectionStore.Unlock()

	log.Printf("Receiver registered with token: %s\n", token)

	// Create a ticker to check connection status every minute
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil {
			log.Println("Failed to set keepalive:", err)
			return
		}
		if err := tcpConn.SetKeepAlivePeriod(time.Second); err != nil {
			log.Println("Failed to set keepalive period:", err)
			return
		}
	}

	for {
		select {
		case <-ticker.C:
			if tcpConn, ok := conn.(*net.TCPConn); !ok || biewire.IsConnClosed(tcpConn) {
				goto cleanup
			}
		}
	}
cleanup:

	// When the receiver disconnects, delete the token
	connectionStore.Lock()
	delete(connectionStore.connections, token)
	connectionStore.Unlock()
	log.Printf("Token expired: %s\n", token)
}

// Forwards sender connection to the receiver and deletes token after first use
func forwardSender(conn net.Conn, cfg Config) {
	defer conn.Close()

	// Extract SNI
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		log.Println("Connection is not TCP")
		return
	}
	serverName, err := biewire.PeekClientHello(tcpConn)
	if serverName == "" || err != nil {
		log.Println("Invalid TLS handshake, no SNI found")
		return
	}
	serverName = strings.ToLower(serverName)

	// Parse `SHARD-ID-XID.relay.com`
	token := strings.Split(serverName, ".")[0]

	// Find receiver connection
	connectionStore.Lock()
	receiverConn, exists := connectionStore.connections[token]
	if !exists {
		connectionStore.Unlock()
		log.Printf("No receiver found for token: %s\n", token)
		return
	}

	// Delete the token immediately after first connection is piped
	delete(connectionStore.connections, token)
	connectionStore.Unlock()
	log.Printf("Token expired after first use: %s\n", token)

	// Forward raw TCP traffic
	log.Printf("Forwarding sender to receiver: %s\n", token)
	pipeConnections(conn, receiverConn)
}

// Pipes two TCP connections together (bi-directional forwarding)
func pipeConnections(conn1, conn2 net.Conn) {
	go func() {
		io.Copy(conn1, conn2)
		conn1.Close()
		conn2.Close()
	}()
	io.Copy(conn2, conn1)
	conn1.Close()
	conn2.Close()
}

func main() {
	cfg, err := env.ParseAs[Config]()
	if err != nil {
		log.Fatalf("Failed to parse environment variables: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setting up logger
	logger := bielog.NewLogger(cfg.LogType, cfg.LogLevel, nil)
	ctx = bielog.CtxWithLogger(ctx, bielog.FromCtx(ctx))

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup

	// Start two listeners - one for senders and one for receivers
	senderListener, err := net.Listen("tcp", net.JoinHostPort("", fmt.Sprintf("%d", cfg.SenderPort)))
	if err != nil {
		logger.ErrorContext(ctx, "Failed to start sender relay server:", err)
		return
	}
	defer senderListener.Close()

	receiverListener, err := net.Listen("tcp", net.JoinHostPort("", fmt.Sprintf("%d", cfg.ReceiverPort)))
	if err != nil {
		logger.ErrorContext(ctx, "Failed to start receiver relay server:", err)
		return
	}
	defer receiverListener.Close()

	logger.InfoContext(ctx, "Relay server running. Sender port: %d, Receiver port: %d\n", cfg.SenderPort, cfg.ReceiverPort)

	// Start receiver handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := receiverListener.Accept()
				if err != nil {
					if !errors.Is(err, net.ErrClosed) {
						logger.ErrorContext(ctx, "Failed to accept receiver connection:", err)
					}
					return
				}
				go registerReceiver(conn, cfg)
			}
		}
	}()

	// Start sender handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := senderListener.Accept()
				if err != nil {
					if !errors.Is(err, net.ErrClosed) {
						logger.ErrorContext(ctx, "Failed to accept sender connection:", err)
					}
					return
				}
				go forwardSender(conn, cfg)
			}
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	logger.InfoContext(ctx, "Shutting down servers...")

	// Initiate graceful shutdown
	cancel()
	senderListener.Close()
	receiverListener.Close()

	// Wait for all connections to finish
	wg.Wait()
	logger.InfoContext(ctx, "Servers stopped gracefully")
}
