package iputil

import (
	"net"
	"net/http"
	"strings"
)

// IPInfo contains IP address classification information
type IPInfo struct {
	IPAddress   string
	IsPrivateIP bool
	IsJapanIP   bool
}

// GetClientIP extracts the client IP address from the HTTP request
// It checks X-Forwarded-For, X-Real-IP headers and falls back to RemoteAddr
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ClassifyIP classifies the IP address
func ClassifyIP(ipAddr string) IPInfo {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return IPInfo{
			IPAddress:   ipAddr,
			IsPrivateIP: false,
			IsJapanIP:   false,
		}
	}

	return IPInfo{
		IPAddress:   ipAddr,
		IsPrivateIP: isPrivateIP(ip),
		IsJapanIP:   isJapanIP(ip),
	}
}

// GetIPInfo is a convenience function that extracts and classifies IP from request
func GetIPInfo(r *http.Request) IPInfo {
	ipAddr := GetClientIP(r)
	return ClassifyIP(ipAddr)
}

// isPrivateIP checks if the IP address is a private IP
func isPrivateIP(ip net.IP) bool {
	// Private IP ranges:
	// 10.0.0.0/8
	// 172.16.0.0/12
	// 192.168.0.0/16
	// 127.0.0.0/8 (loopback)
	// ::1/128 (IPv6 loopback)
	// fc00::/7 (IPv6 private)

	privateIPBlocks := []*net.IPNet{
		{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
		{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)},
		{IP: net.ParseIP("127.0.0.0"), Mask: net.CIDRMask(8, 32)},
		{IP: net.ParseIP("::1"), Mask: net.CIDRMask(128, 128)},
		{IP: net.ParseIP("fc00::"), Mask: net.CIDRMask(7, 128)},
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}

	return false
}

// isJapanIP checks if the IP address is from Japan
// This is a simplified implementation using common Japanese IP ranges
// For production, consider using a GeoIP database like MaxMind GeoLite2
func isJapanIP(ip net.IP) bool {
	// Sample Japanese IP ranges (major ISPs and cloud providers in Japan)
	// This is a simplified list - in production, use a proper GeoIP database
	japanIPRanges := []string{
		// NTT
		"1.0.16.0/20",
		"1.0.64.0/18",
		"1.1.0.0/16",
		"1.21.0.0/16",
		"1.33.0.0/16",
		// KDDI
		"27.80.0.0/12",
		"49.96.0.0/11",
		"60.32.0.0/11",
		// SoftBank
		"61.192.0.0/12",
		"114.48.0.0/13",
		"126.0.0.0/8",
		// IIJ
		"202.232.0.0/13",
		// AWS Tokyo Region (sample)
		"13.112.0.0/14",
		"13.230.0.0/15",
		"18.176.0.0/13",
		// Google Cloud Tokyo (sample)
		"34.84.0.0/14",
		"35.187.192.0/19",
		"35.189.128.0/17",
		// Azure Japan (sample)
		"20.43.64.0/18",
		"20.189.0.0/18",
		"40.74.0.0/16",
	}

	for _, cidr := range japanIPRanges {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if block.Contains(ip) {
			return true
		}
	}

	return false
}
