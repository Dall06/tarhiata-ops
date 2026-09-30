package repositories

import "net"

// NetDNSResolver implementa ports.DNSResolver usando el resolutor DNS real del sistema.
type NetDNSResolver struct{}

func NewNetDNSResolver() *NetDNSResolver {
	return &NetDNSResolver{}
}

func (*NetDNSResolver) LookupHost(host string) ([]string, error) {
	return net.LookupHost(host)
}
