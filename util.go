package tender

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

var portMap = map[string]string{
	"http":  "80",
	"https": "443",
}

func isWebSocket(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.EqualFold(r.Header.Get("Connection"), "upgrade")
}

// parsePort returns the port of an HTTP request. If there is no port specified
// in the request, 80 or 443 (HTTPS) is assumed.
func parsePort(req *http.Request) string {
	if _, p, err := net.SplitHostPort(req.Host); err == nil {
		if p == "443" {
			return req.FormValue("freighterd_port")
		} else {
			return p
		}
	}
	return portMap[req.URL.Scheme]
}

// ConvertToIPBased modifies an HTTP request to use an IP address as its host.
// The "Host" header is preserved so that receiver will correctly recognize the
// original host instead of the ip address
func hostToIP(req *http.Request, ip net.IP) string {
	// Change Host to "IP:Port".
	port := parsePort(req)
	h := net.JoinHostPort(ip.String(), port)
	req.URL.Host = h
	return h
}

// Given a request with host:
// winterfell.yammer.nitrousapp.com:8000, it will return host name as "yammer"
// yammer.nitrousapp.com:65333, it will return host name as "yammer"
func getHost(url, baseDomain string) (string, error) {
	var domain string
	var err error
	if strings.Contains(url, ":") {
		domain, _, err = net.SplitHostPort(url)
		if err != nil {
			return "", err
		}
	} else {
		domain = url
	}

	prefix := domain[:len(domain)-len(baseDomain)]
	splits := strings.Split(prefix, ".")
	// request only specified host
	if len(splits) == 1 {
		return splits[0], nil
	}
	// request specified both container and host
	if len(splits) == 2 {
		return splits[1], nil
	}
	return "", fmt.Errorf("Invalid format %s", url)
}
