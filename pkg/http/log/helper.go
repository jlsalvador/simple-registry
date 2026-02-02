package log

import (
	"net"
	"net/http"
	"strings"
)

func GetClientIP(r *http.Request) string {
	// Helper to clean IP (remove zone for IPv6)
	cleanIP := func(ip string) string {
		ip = strings.Split(ip, "%")[0] // remove IPv6 zone if present
		if net.ParseIP(ip) != nil {
			return ip
		}
		return ""
	}

	// 1. X-Forwarded-For (first IP, trusted proxy only)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := cleanIP(strings.TrimSpace(parts[0])); ip != "" {
			return ip
		}
	}

	// 2. X-Real-IP
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		if ip := cleanIP(xrip); ip != "" {
			return ip
		}
	}

	// 3. RemoteAddr (strip port)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := cleanIP(host); ip != "" {
		return ip
	}

	return ""
}
