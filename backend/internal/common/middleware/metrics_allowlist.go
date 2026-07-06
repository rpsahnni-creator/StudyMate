package middleware

import (
	"net"
	"net/http"
	"os"
	"strings"
)

// MetricsAllowlist restricts /metrics to configured IP ranges (internal scrapers).
func MetricsAllowlist(next http.Handler) http.Handler {
	allowed := loadMetricsAllowedIPs()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(allowed) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		ip := ClientIP(r)
		if !ipAllowed(ip, allowed) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loadMetricsAllowedIPs() []*net.IPNet {
	raw := strings.TrimSpace(os.Getenv("METRICS_ALLOWED_IPS"))
	if raw == "" {
		if strings.EqualFold(os.Getenv("ENVIRONMENT"), "production") {
			return defaultPrivateCIDRs()
		}
		return nil
	}
	var nets []*net.IPNet
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			_, n, err := net.ParseCIDR(part)
			if err == nil {
				nets = append(nets, n)
			}
			continue
		}
		ip := net.ParseIP(part)
		if ip == nil {
			continue
		}
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	return nets
}

func defaultPrivateCIDRs() []*net.IPNet {
	cidrs := []string{
		"127.0.0.1/32",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
	var out []*net.IPNet
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err == nil {
			out = append(out, n)
		}
	}
	return out
}

func ipAllowed(ipStr string, nets []*net.IPNet) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}
