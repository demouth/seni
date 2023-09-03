package seni

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
)

// Test creates a readWriter and calls ServeConn on local servver
func (s *Seni) Test(req *http.Request) (*http.Response, error) {
	d, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil, err
	}
	raw := string(d)
	s.init()
	rw := &readWriter{}
	rw.r.WriteString(raw)

	channel := make(chan error)
	go func() {
		channel <- s.server.ServeConn(rw)
	}()

	select {
	case err := <-channel:
		if err != nil {
			return nil, err
		}
	case <-time.After(200 * time.Millisecond):
		return nil, fmt.Errorf("Timeout")
	}

	// Read response
	buffer := bufio.NewReader(&rw.w)

	// Convert raw http response to *http.Response
	res, err := http.ReadResponse(buffer, req)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return res, nil
}

// TrimRightBytes is the equivalent of bytes.TrimRight
func TrimRightBytes(b []byte, cutset byte) []byte {
	lenStr := len(b)
	for lenStr > 0 && b[lenStr-1] == cutset {
		lenStr--
	}
	return b[:lenStr]
}

func TrimRight(s string, cutset byte) string {
	lenStr := len(s)
	for lenStr > 0 && s[lenStr-1] == cutset {
		lenStr--
	}
	return s[:lenStr]
}

// Readwriter for test cases
type readWriter struct {
	net.Conn
	r bytes.Buffer
	w bytes.Buffer
}

func (rw *readWriter) Close() error {
	return nil
}

func (rw *readWriter) Read(b []byte) (int, error) {
	return rw.r.Read(b)
}

func (rw *readWriter) Write(b []byte) (int, error) {
	return rw.w.Write(b)
}

func (rw *readWriter) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP: net.IPv4zero,
	}
}

func (rw *readWriter) LocalAddr() net.Addr {
	return rw.RemoteAddr()
}

func (rw *readWriter) SetReadDeadline(t time.Time) error {
	return nil
}

func (rw *readWriter) SetWriteDeadline(t time.Time) error {
	return nil
}
