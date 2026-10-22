package discovery

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/hashicorp/mdns"
)

// ServiceType is the DNS-SD service type advertised in LAN (client browses the same).
const ServiceType = "_buhgalter._tcp"

// Advertiser publishes the Buhgalter HTTP API via mDNS.
type Advertiser struct {
	server *mdns.Server
	logger *slog.Logger
}

type AdvertiseConfig struct {
	Addr         string
	InstanceName string
	Version      string
	Hostname     string
	LocalIPv4    []net.IP
}

// NewAdvertiser builds an mDNS advertiser; call Start to publish.
func NewAdvertiser(cfg AdvertiseConfig, logger *slog.Logger) (*Advertiser, error) {
	if logger == nil {
		logger = slog.Default()
	}

	port, err := ParseHTTPPort(cfg.Addr)
	if err != nil {
		return nil, fmt.Errorf("mdns port: %w", err)
	}

	ips := cfg.LocalIPv4
	if len(ips) == 0 {
		ips = localIPv4Addresses()
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("mdns: no local IPv4 addresses")
	}

	instance := strings.TrimSpace(cfg.InstanceName)
	if instance == "" {
		instance = "Buhgalter"
	}

	host := strings.TrimSpace(cfg.Hostname)
	if host == "" {
		host, _ = os.Hostname()
	}
	if host == "" {
		host = "buhgalter"
	}
	hostLabel := host + ".local."

	version := strings.TrimSpace(cfg.Version)
	txt := []string{
		"app=buhgalter",
		"path=/api/v1/health",
	}
	if version != "" {
		txt = append(txt, "version="+version)
	}

	service, err := mdns.NewMDNSService(instance, ServiceType, "local.", hostLabel, port, ips, txt)
	if err != nil {
		return nil, fmt.Errorf("mdns service: %w", err)
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		return nil, fmt.Errorf("mdns server: %w", err)
	}

	return &Advertiser{server: server, logger: logger}, nil
}

func (a *Advertiser) Start() error {
	if a == nil || a.server == nil {
		return fmt.Errorf("mdns: not configured")
	}
	a.logger.Info("mdns advertising", "service", ServiceType)
	return nil
}

func (a *Advertiser) Stop() {
	if a == nil || a.server == nil {
		return
	}
	_ = a.server.Shutdown()
	a.server = nil
}

func localIPv4Addresses() []net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var ips []net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.To4() == nil || ip.IsLoopback() {
				continue
			}
			ips = append(ips, ip)
		}
	}
	return ips
}
