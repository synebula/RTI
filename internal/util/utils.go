package util

import (
	crand "crypto/rand"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func RandomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(crand.Reader, buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func DetectLocalIP() string {
	conn, err := net.Dial("udp", "192.0.2.1:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	if udpAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return udpAddr.IP.String()
	}
	return "127.0.0.1"
}

func QuotedClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return host
}

func FormatTextPreview(text string, limit int) string {
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "\r", "\\r"), "\n", "\\n")
	if len(normalized) <= limit {
		return normalized
	}
	return normalized[:limit-3] + "..."
}

func MaxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
