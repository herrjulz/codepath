package server

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func loopbackHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func allowRequest(w http.ResponseWriter, r *http.Request) bool {
	if !loopbackHost(r.Host) {
		writeError(w, http.StatusForbidden, "forbidden_host", "Host is not a loopback address.", nil)
		return false
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || !strings.EqualFold(u.Host, r.Host) || (u.Scheme != "http" && u.Scheme != "https") {
			writeError(w, http.StatusForbidden, "forbidden_origin", "Origin is not allowed.", nil)
			return false
		}
	}
	return true
}

func authorized(r *http.Request, token string) bool {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		subtle.ConstantTimeCompare([]byte(token), []byte(token))
		return false
	}
	got := header[len(prefix):]
	if len(got) != len(token) {
		subtle.ConstantTimeCompare([]byte(token), []byte(token))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}
