package pinger

import (
	"errors"
	"net"

	"golang.org/x/net/icmp"
)

// Get the appropriate address based on the ICMP connection type
func getAddr(c *icmp.PacketConn, addr string) (net.IP, net.Addr, error) {
	ips, err := net.LookupIP(addr)
	if err != nil {
		return nil, nil, err
	}
	netaddr := func(ip net.IP) (net.Addr, error) {
		switch c.LocalAddr().(type) {
		case *net.UDPAddr:
			return &net.UDPAddr{IP: ip}, nil
		case *net.IPAddr:
			return &net.IPAddr{IP: ip}, nil
		default:
			// This should never happen, since we control the connection type and
			// we never expose this to library users.
			return nil, errors.New("Cannot determine connection type")
		}
	}
	for _, ip := range ips {
		addr, err := netaddr(ip)
		return ip, addr, err
	}
	return nil, nil, errors.New("Cannot resolve address")
}
