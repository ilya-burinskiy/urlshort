package services

import (
	"net"

	"github.com/ilya-burinskiy/urlshort/internal/app/configs"
)

type IPChecker interface {
	InTrustedSubnet(ip net.IP) bool
}

func NewIPChecker(config configs.Config) IPChecker {
	return ipChecker{
		config: config,
	}
}

type ipChecker struct {
	config configs.Config
}

func (c ipChecker) InTrustedSubnet(ip net.IP) bool {
	_, ipv4Net, err := net.ParseCIDR(c.config.TrustedSubnet)
	if err != nil {
		return false
	}

	return ipv4Net.Contains(ip)
}
