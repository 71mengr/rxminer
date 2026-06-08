package pool

import (
	"fmt"
	"net"
)

func ExternalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP != nil && !addr.IP.IsLoopback() {
			if ip := addr.IP.To4(); ip != nil {
				return ip.String()
			}
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

		if ip == nil || ip.IsLoopback() {
			continue
		}

		if ip = ip.To4(); ip == nil {
			continue
		}

		return ip.String()
	}

	return "0.0.0.0"
}

func ExternalWebURL(port int) string {
	return fmt.Sprintf("http://%s:%d", ExternalIP(), port)
}
