package config

import (
	"crypto/sha256"
	"crypto/subtle"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port           string
	DataDir        string
	StaticDir      string
	AdminSecret    string
	PublicURL      string
	PublicView     string // "scene-only", "full", "private"
	TrustedProxies []*net.IPNet
	TZ             *time.Location
	RetentionDays  int
	DemoMode       bool
	MetricsToken   string
}

func Load() (*Config, error) {
	port := getEnv("PORT", "7437")
	dataDir := getEnv("DATA_DIR", "./data")
	staticDir := getEnv("STATIC_DIR", "./web/dist")
	adminSecret := os.Getenv("ADMIN_SECRET")
	publicURL := getEnv("PUBLIC_URL", "http://localhost:"+port)
	publicView := getEnv("PUBLIC_VIEW", "scene-only")
	metricsToken := os.Getenv("METRICS_TOKEN")

	retentionDays, _ := strconv.Atoi(getEnv("RETENTION_DAYS", "395"))
	if retentionDays <= 0 {
		retentionDays = 395
	}

	demoMode := strings.EqualFold(os.Getenv("DEMO_MODE"), "true")

	// Timezone
	tzName := getEnv("TZ", "Asia/Jakarta")
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		loc = time.UTC
	}

	// Trusted proxies
	trustedRaw := getEnv("TRUSTED_PROXIES", "127.0.0.1/32,::1/128,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")
	var trustedNets []*net.IPNet
	for _, cidr := range strings.Split(trustedRaw, ",") {
		cidr = strings.TrimSpace(cidr)
		if cidr == "" {
			continue
		}
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		_, ipNet, err := net.ParseCIDR(cidr)
		if err == nil {
			trustedNets = append(trustedNets, ipNet)
		}
	}

	return &Config{
		Port:           port,
		DataDir:        dataDir,
		StaticDir:      staticDir,
		AdminSecret:    adminSecret,
		PublicURL:      publicURL,
		PublicView:     publicView,
		TrustedProxies: trustedNets,
		TZ:             loc,
		RetentionDays:  retentionDays,
		DemoMode:       demoMode,
		MetricsToken:   metricsToken,
	}, nil
}

func (c *Config) ValidateAdminSecret(provided string) bool {
	if c.AdminSecret == "" || provided == "" {
		return false
	}
	hashExpected := sha256.Sum256([]byte(c.AdminSecret))
	hashProvided := sha256.Sum256([]byte(provided))
	return subtle.ConstantTimeCompare(hashExpected[:], hashProvided[:]) == 1
}

func (c *Config) IsTrustedProxy(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, network := range c.TrustedProxies {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	return val
}
