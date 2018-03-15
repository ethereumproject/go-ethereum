package distip

import (
	"bytes"
	"fmt"
	"net"
	"sort"
)

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
	key := s.key(ip)
	n := s.members[string(key)]
	if n < s.Limit {
		s.members[string(key)] = n + 1
		return true
	}
	return false
}

// Remove removes an IP from the set.
func (s *DistinctNetSet) Remove(ip net.IP) {
	key := s.key(ip)
	if n, ok := s.members[string(key)]; ok {
		if n == 1 {
			delete(s.members, string(key))
		} else {
			s.members[string(key)] = n - 1
		}
	}
}

// Contains whether the given IP is contained in the set.
func (s DistinctNetSet) Contains(ip net.IP) bool {
	key := s.key(ip)
	_, ok := s.members[string(key)]
	return ok
}

// Len returns the number of tracked IPs.
func (s DistinctNetSet) Len() int {
	n := uint(0)
	for _, i := range s.members {
		n += i
	}
	return int(n)
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
