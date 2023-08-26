package common

import (
	"net"
)

type IP struct {
	Scheme string `json:"schema"`
	IPv4   net.IP `json:"ipv4"`
	Port   string `json:"port"`
}

func (ip IP) String() string {
	return ip.Scheme + "://" + ip.IPv4.String() + ":" + ip.Port
}
