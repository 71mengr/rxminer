package pool

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var publicIPServices = []string{
	"https://api.ipify.org",
	"https://ifconfig.me/ip",
	"https://icanhazip.com",
}

func ExternalIP() string {
	if ip := publicIPFromServices(publicIPServices); ip != "" {
		return ip
	}

	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP != nil && isPublicIPv4(addr.IP) {
			return addr.IP.To4().String()
		}
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "0.0.0.0"
	}

	for _, addr := range addrs {
		var ip net.IP
		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if !isPublicIPv4(ip) {
			continue
		}

		return ip.To4().String()
	}

	return "0.0.0.0"
}

func publicIPFromServices(services []string) string {
	client := http.Client{Timeout: 2 * time.Second}

	for _, service := range services {
		resp, err := client.Get(service)
		if err != nil {
			continue
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if err != nil || resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			continue
		}

		ip := parsePublicIPv4(string(body))
		if ip != "" {
			return ip
		}
	}

	return ""
}

func parsePublicIPv4(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if !isPublicIPv4(ip) {
		return ""
	}

	return ip.To4().String()
}

func isPublicIPv4(ip net.IP) bool {
	if ip == nil {
		return false
	}

	ip = ip.To4()
	if ip == nil {
		return false
	}

	return !ip.IsUnspecified() && !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast()
}

func ExternalWebURL(port int) string {
	return fmt.Sprintf("http://%s:%d", ExternalIP(), port)
}
