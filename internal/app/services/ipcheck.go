package services

import (
	"net"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
)

type IPChecker struct {
	config configs.Config
}

func NewIPChecker(config configs.Config) IPChecker {
	return IPChecker{
		config: config,
	}
}

func (c IPChecker) InTrustedSubnet(ip net.IP) bool {
	_, ipv4Net, err := net.ParseCIDR(c.config.TrustedSubnet)
	if err != nil {
		return false
	}

	return ipv4Net.Contains(ip)
}
