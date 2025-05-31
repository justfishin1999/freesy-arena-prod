// network/netscan.go

package network

import (
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Device struct {
	IP       string    `json:"ip"`
	LastSeen time.Time `json:"lastSeen"`
	Rogue    bool      `json:"rogue"`
}

type scanner struct {
	Subnet       *net.IPNet
	Interval     time.Duration
	Devices      map[string]Device
	KnownDevices map[string]bool
	Mutex        sync.Mutex
}

type Manager struct {
	Enabled bool
	scanner *scanner
}

var managerInstance *Manager = &Manager{}

// GetManager returns the singleton Manager.
func GetManager() *Manager {
	return managerInstance
}

// SetEnabled toggles the scanner on/off. If enabling, it passes both the subnet string
// and the desired scan interval (in seconds). If no scanner is running yet, it will attempt
// to parse the CIDR (appending "/24" if needed) and start the background loop.
func (m *Manager) SetEnabled(enabled bool, subnet string, configuredInterval int) {
	m.Enabled = enabled
	if enabled && m.scanner == nil {
		m.start(subnet, configuredInterval)
	}
	// NOTE: We do not cancel the existing scan goroutine if enabled==false. To fully stop it,
	// you would need a context or a stop channel—this example leaves it running (but it will
	// simply not be used if Enabled==false).
}

// start parses the given CIDR (adding "/24" if missing), creates a new scanner with the given interval,
// and spins up the background ping loop.
func (m *Manager) start(cidr string, configuredInterval int) {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		log.Printf("Network scanner not started: no subnet configured.\n")
		return
	}

	// If the user omitted a mask (e.g. "10.0.100.0"), default to /24
	if !strings.Contains(cidr, "/") {
		cidr = cidr + "/24"
	}

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		log.Printf("Network scanner not started: invalid CIDR \"%s\": %v\n", cidr, err)
		return
	}

	// Build a new scanner with the parsed subnet and the interval provided (in seconds).
	sc := &scanner{
		Subnet:       ipnet,
		Interval:     time.Duration(configuredInterval) * time.Second,
		Devices:      make(map[string]Device),
		KnownDevices: map[string]bool{"10.0.100.5": true, "10.0.100.3": true, "10.0.100.2": true},
	}
	m.scanner = sc

	// Launch the background goroutine that continuously scans.
	go func() {
		for {
			sc.scan()
			time.Sleep(sc.Interval)
		}
	}()
	log.Printf("Network scanner started on subnet %s (interval %ds)\n", cidr, configuredInterval)
}

// GetDevices returns the slice of last‐seen devices. If no scan has ever started, returns nil.
func (m *Manager) GetDevices() []Device {
	if m.scanner == nil {
		return nil
	}
	m.scanner.Mutex.Lock()
	defer m.scanner.Mutex.Unlock()

	var list []Device
	for _, d := range m.scanner.Devices {
		list = append(list, d)
	}
	return list
}

// scan performs an ICMP ping sweep of every IP in the configured subnet.
// If an IP is reachable, it records it (updating LastSeen and Rogue).
func (s *scanner) scan() {
	for ip := s.Subnet.IP.Mask(s.Subnet.Mask); s.Subnet.Contains(ip); incrementIP(ip) {
		addr := ip.String()
		if reachable, _ := ping(addr); reachable {
			s.Devices[addr] = Device{
				IP:       addr,
				LastSeen: time.Now(),
				Rogue:    !s.KnownDevices[addr],
			}
		}
	}
}

// incrementIP moves the given IP to the next address in network order.
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

// ping sends a single ICMP Echo Request to the given IP, waiting up to 500ms for a reply.
func ping(ip string) (bool, error) {
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false, err
	}
	defer c.Close()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   os.Getpid() & 0xffff,
			Seq:  1,
			Data: []byte("cheesy-scan"),
		},
	}
	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return false, err
	}

	dst, err := net.ResolveIPAddr("ip4", ip)
	if err != nil {
		return false, err
	}

	start := time.Now()
	if _, err = c.WriteTo(msgBytes, dst); err != nil {
		return false, err
	}

	reply := make([]byte, 1500)
	if err := c.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		return false, err
	}

	n, peer, err := c.ReadFrom(reply)
	if err != nil {
		// timed out or unreachable
		return false, nil
	}

	parsedMsg, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply[:n])
	if err != nil {
		return false, err
	}

	if parsedMsg.Type == ipv4.ICMPTypeEchoReply {
		log.Printf("Network Scanner: Reply from %s in %v", peer.String(), time.Since(start))
		return true, nil
	}
	return false, nil
}

// MarkAsKnown flags the given IP address as a known device. Any existing entry in s.Devices[ip].Rogue
// will be set to false, and future scans will not mark it as rogue.
func (m *Manager) MarkAsKnown(ip string) {
	if m.scanner == nil {
		return
	}
	m.scanner.Mutex.Lock()
	defer m.scanner.Mutex.Unlock()

	// Add to known devices so future scans won't mark it as rogue
	m.scanner.KnownDevices[ip] = true

	// If it’s already in Devices map, update its Rogue field immediately
	if d, ok := m.scanner.Devices[ip]; ok {
		d.Rogue = false
		m.scanner.Devices[ip] = d
	}
}
