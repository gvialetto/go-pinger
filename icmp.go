package pinger

import (
	"net"
	"syscall"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	// IANA defined, unfortunately the "golang.org/x/net/internal/iana" package
	// is internal and we cannot use it directly, so we have to declare this
	// here again.
	protocolICMPv4 = 1
	// This is the protocol we can use if we have root access (or
	// CAP_NET_RAW is set, with `setcap cap_net_raw+ep` for example)
	icmpProtoICMP4 = "ip4:icmp"
	// This is the protocol we are forced to use if we don't have privileged
	// access. The net.ipv4.ping_group_range sysctl needs to be set up
	// appropriately though.
	icmpProtoUDP4 = "udp4"
)

// Simple alias to return a listener to 0.0.0.0 (all interfaces)
func getDefaultICMPListener() (*icmp.PacketConn, string, error) {
	return getICMPListener("0.0.0.0")
}

func getICMPListener(addr string) (*icmp.PacketConn, string, error) {
	// We have two choices here. If we're running as root or the application has
	// CAP_NET_RAW we can use ICMP directly. Otherwise we need to fall back to
	// UDP/ICMP sockets, but that may still fail since it requires the
	// net.ipv4.ping_group_range to be configured properly.
	// There is little point in trying to detect in which case we're falling in,
	// so just try until we succeed (and if we don't, error out)
	var retErr error
	protocols := []string{icmpProtoICMP4, icmpProtoUDP4}
	for _, proto := range protocols {
		conn, err := icmp.ListenPacket(proto, addr)
		if err == nil {
			return conn, proto, err
		}
		retErr = err
	}
	return nil, "", retErr
}

func sendICMPEchoMessage(
	conn *icmp.PacketConn,
	addr net.Addr,
	srcID int,
	seq int,
	data []byte,
) error {
	wm := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: srcID, Seq: seq,
			Data: data,
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}
	for {
		if _, err := conn.WriteTo(wb, addr); err != nil {
			if neterr, ok := err.(*net.OpError); ok {
				// keep writing until we're done
				if neterr.Err == syscall.ENOBUFS {
					continue
				}
			}
		}
		break
	}
	return nil
}

func parseICMPMessage(data []byte) *icmp.Echo {
	m, err := icmp.ParseMessage(protocolICMPv4, data)
	if err != nil {
		return nil
	}
	// If we are not interested into the packet just ignore it
	if m.Type != ipv4.ICMPTypeEchoReply {
		return nil
	}
	switch messageType := m.Body.(type) {
	case *icmp.Echo:
		return messageType
	}
	return nil
}
