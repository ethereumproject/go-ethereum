package distip

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sort"
)

var (
	lan4, lan6, special4, special6 Netlist
	errInvalid                     = errors.New("invalid IP")
	errUnspecified                 = errors.New("zero address")
	errSpecial                     = errors.New("special network")
	errLoopback                    = errors.New("loopback address from non-loopback host")
	errLAN                         = errors.New("LAN address from WAN host")
)

// Netlist is a list of IP networks.
type Netlist []net.IPNet

func init() {
	// Lists from RFC 5735, RFC 5156,
	// https://www.iana.org/assignments/iana-ipv4-special-registry/
	lan4.Add("0.0.0.0/8")              // "This" network
	lan4.Add("10.0.0.0/8")             // Private Use
	lan4.Add("172.16.0.0/12")          // Private Use
	lan4.Add("192.168.0.0/16")         // Private Use
	lan6.Add("fe80::/10")              // Link-Local
	lan6.Add("fc00::/7")               // Unique-Local
	special4.Add("192.0.0.0/29")       // IPv4 Service Continuity
	special4.Add("192.0.0.9/32")       // PCP Anycast
	special4.Add("192.0.0.170/32")     // NAT64/DNS64 Discovery
	special4.Add("192.0.0.171/32")     // NAT64/DNS64 Discovery
	special4.Add("192.0.2.0/24")       // TEST-NET-1
	special4.Add("192.31.196.0/24")    // AS112
	special4.Add("192.52.193.0/24")    // AMT
	special4.Add("192.88.99.0/24")     // 6to4 Relay Anycast
	special4.Add("192.175.48.0/24")    // AS112
	special4.Add("198.18.0.0/15")      // Device Benchmark Testing
	special4.Add("198.51.100.0/24")    // TEST-NET-2
	special4.Add("203.0.113.0/24")     // TEST-NET-3
	special4.Add("255.255.255.255/32") // Limited Broadcast

	// http://www.iana.org/assignments/iana-ipv6-special-registry/
	special6.Add("100::/64")
	special6.Add("2001::/32")
	special6.Add("2001:1::1/128")
	special6.Add("2001:2::/48")
	special6.Add("2001:3::/32")
	special6.Add("2001:4:112::/48")
	special6.Add("2001:5::/32")
	special6.Add("2001:10::/28")
	special6.Add("2001:20::/28")
	special6.Add("2001:db8::/32")
	special6.Add("2002::/16")
}

// Add parses a CIDR mask and appends it to the list. It panics for invalid masks and is
// intended to be used for setting up static lists.
func (l *Netlist) Add(cidr string) {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	*l = append(*l, *n)
}

// Contains reports whether the given IP is contained in the list.
func (l *Netlist) Contains(ip net.IP) bool {
	if l == nil {
		return false
	}
	for _, net := range *l {
		if net.Contains(ip) {
			return true
		}
	}
	return false
}

// IsLAN reports whether an IP is a local network address.
func IsLAN(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		return lan4.Contains(v4)
	}
	return lan6.Contains(ip)
}

// IsSpecialNetwork reports whether an IP is located in a special-use network range
// This includes broadcast, multicast and documentation addresses.
func IsSpecialNetwork(ip net.IP) bool {
	if ip.IsMulticast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil {
		return special4.Contains(v4)
	}
	return special6.Contains(ip)
}

// CheckRelayIP reports whether an IP relayed from the given sender IP
// is a valid connection target.
//
// There are four rules:
//   - Special network addresses are never valid.
//   - Loopback addresses are OK if relayed by a loopback host.
//   - LAN addresses are OK if relayed by a LAN host.
//   - All other addresses are always acceptable.
func CheckRelayIP(sender, addr net.IP) error {
	if len(addr) != net.IPv4len && len(addr) != net.IPv6len {
		return errInvalid
	}
	if addr.IsUnspecified() {
		return errUnspecified
	}
	if IsSpecialNetwork(addr) {
		return errSpecial
	}
	if addr.IsLoopback() && !sender.IsLoopback() {
		return errLoopback
	}
	if IsLAN(addr) && !IsLAN(sender) {
		return errLAN
	}
	return nil
}

// DistinctNetSet tracks IPs, ensuring that at most N of them
// fall into the same network range.
type DistinctNetSet struct {
	Subnet uint // number of common prefix bits
	Limit  uint // maximum number of IPs in each subnet

	members map[string]uint
	buf     net.IP
}

// Add adds an IP address to the set. It returns false (and doesn't add the IP) if the
// number of existing IPs in the defined range exceeds the limit.
func (s *DistinctNetSet) Add(ip net.IP) bool {
	key := string(s.key(ip))
	n := s.members[key]
	if n < s.Limit {
		s.members[key] = n + 1
		return true
	}
	return false
}

// Remove removes an IP from the set.
func (s *DistinctNetSet) Remove(ip net.IP) {
	key := string(s.key(ip))
	if n, ok := s.members[key]; ok {
		if n == 1 {
			delete(s.members, key)
		} else {
			s.members[key] = n - 1
		}
	}
}

// Contains whether the given IP is contained in the set.
func (s DistinctNetSet) Contains(ip net.IP) bool {
	key := string(s.key(ip))
	_, ok := s.members[key]
	return ok
}

// Len returns the number of tracked IPs.
func (s DistinctNetSet) Len() uint {
	n := uint(0)
	for _, i := range s.members {
		n += i
	}
	return n
}

// key encodes the map key for an address into a temporary buffer.
//
// The first byte of key is '4' or '6' to distinguish IPv4/IPv6 address types.
// The remainder of the key is the IP, truncated to the number of bits.
func (s *DistinctNetSet) key(ip net.IP) net.IP {
	// Lazily initialize storage.
	if s.members == nil {
		s.members = make(map[string]uint)
		s.buf = make(net.IP, 17)
	}
	// Canonicalize ip and bits.
	typ := byte('6')
	if ip4 := ip.To4(); ip4 != nil {
		typ, ip = '4', ip4
	}
	bits := s.Subnet
	if bits > uint(len(ip)*8) {
		bits = uint(len(ip) * 8)
	}
	// Encode the prefix into s.buf.
	nb := int(bits / 8)
	mask := ^byte(0xFF >> (bits % 8))
	s.buf[0] = typ
	buf := append(s.buf[:1], ip[:nb]...)
	if nb < len(ip) && mask != 0 {
		buf = append(buf, ip[nb]&mask)
	}
	return buf
}

// String implements fmt.Stringer
func (s DistinctNetSet) String() string {
	var buf bytes.Buffer
	buf.WriteString("{")
	keys := make([]string, 0, len(s.members))
	for k := range s.members {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		var ip net.IP
		if k[0] == '4' {
			ip = make(net.IP, 4)
		} else {
			ip = make(net.IP, 16)
		}
		copy(ip, k[1:])
		fmt.Fprintf(&buf, "%v×%d", ip, s.members[k])
		if i != len(keys)-1 {
			buf.WriteString(" ")
		}
	}
	buf.WriteString("}")
	return buf.String()
}
