package osserver

import (
	"context"
	"log"
	"net"
	"net/http"
)

// Listener

type singleConnListener struct {
	conn     net.Conn
	doneChan chan struct{}
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.conn == nil {
		select {
		case <-l.doneChan:
			return nil, net.ErrClosed
		}
	}
	conn := l.conn
	l.conn = nil
	return conn, nil
}

func (l *singleConnListener) Close() error {
	l.doneChan <- struct{}{}
	return nil
}

func (l *singleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

// Server

type OneShotServer struct {
	conn net.Conn
	mux  *http.ServeMux
	srv  *http.Server
	done chan bool
}

func NewOneShotServer(conn net.Conn, mux *http.ServeMux) *OneShotServer {
	return &OneShotServer{
		conn: conn,
		mux:  mux,
		done: make(chan bool),
	}
}

func (s *OneShotServer) Serve(ctx context.Context) error {
	listener := &singleConnListener{conn: s.conn, doneChan: make(chan struct{})}

	s.srv = &http.Server{
		Handler: doneMiddleware(s.mux, s.done),
	}
	go func() {
		if err := s.srv.Serve(listener); err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
		s.done <- true
	}()

	<-s.done
	// Gracefull shutdown
	return s.srv.Shutdown(ctx)
}

func doneMiddleware(next http.Handler, done chan bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		done <- true
	})
}
