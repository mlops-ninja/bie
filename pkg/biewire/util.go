package biewire

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func IsConnClosed(conn *net.TCPConn) bool {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return true // Assume closed if we can't get raw connection.
	}

	closed := false
	err = rawConn.Control(func(fd uintptr) {
		pollFd := []unix.PollFd{
			{Fd: int32(fd), Events: unix.POLLIN | unix.POLLHUP | unix.POLLERR},
		}

		_, err := unix.Poll(pollFd, 0)
		if err != nil || pollFd[0].Revents&(unix.POLLHUP|unix.POLLERR) != 0 {
			closed = true
		}
	})
	if err != nil {
		return true // Treat any syscall error as closed.
	}
	return closed
}

// Peek the first few bytes of the handshake
func PeekClientHello(conn *net.TCPConn) (string, error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return "", err
	}

	var sni string
	err = rawConn.Control(func(fd uintptr) {
		var errno error
		var n int
		buf := make([]byte, 1024) // Large enough for the ClientHello
		for {
			n, _, errno = syscall.Recvfrom(int(fd), buf, syscall.MSG_PEEK)
			if errno == syscall.EWOULDBLOCK || n == 0 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			break
		}

		if errno != nil || n < 5 {
			return
		}

		// Parse TLS ClientHello
		sni, _ = extractSNI(buf[:n])
	})
	if err != nil {
		return "", err
	}
	if sni == "" {
		return "", errors.New("SNI not found")
	}
	return sni, nil
}

// Extract SNI from ClientHello
func extractSNI(data []byte) (string, error) {
	helloInfo, err := readClientHello(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	return helloInfo.ServerName, nil
}

func readClientHello(reader io.Reader) (*tls.ClientHelloInfo, error) {
	var hello *tls.ClientHelloInfo

	err := tls.Server(readOnlyConn{reader: reader}, &tls.Config{
		GetConfigForClient: func(argHello *tls.ClientHelloInfo) (*tls.Config, error) {
			hello = new(tls.ClientHelloInfo)
			*hello = *argHello
			return nil, nil
		},
	}).Handshake()

	if hello == nil {
		return nil, err
	}

	return hello, nil
}

type readOnlyConn struct {
	reader io.Reader
}

func (conn readOnlyConn) Read(p []byte) (int, error)         { return conn.reader.Read(p) }
func (conn readOnlyConn) Write(p []byte) (int, error)        { return 0, io.ErrClosedPipe }
func (conn readOnlyConn) Close() error                       { return nil }
func (conn readOnlyConn) LocalAddr() net.Addr                { return nil }
func (conn readOnlyConn) RemoteAddr() net.Addr               { return nil }
func (conn readOnlyConn) SetDeadline(t time.Time) error      { return nil }
func (conn readOnlyConn) SetReadDeadline(t time.Time) error  { return nil }
func (conn readOnlyConn) SetWriteDeadline(t time.Time) error { return nil }
