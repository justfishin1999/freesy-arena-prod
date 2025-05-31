// network/netscan.go

package network

import (
	"fmt"
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

func GetManager() *Manager {
	return managerInstance
}

// SetEnabled toggles the scanner on/off and, if enabling, passes the subnet string.
// If `enabled==true` and no scanner is running, it will attempt to parse the CIDR (adding "/24" if needed).
func (m *Manager) SetEnabled(enabled bool, subnet string) {
	m.Enabled = enabled
	if enabled && m.scanner == nil {
		m.start(subnet)
	}
	// (Note: We are not explicitly stopping the existing goroutine here if enabled==false.
	//  If you need to fully cancel/stop the background loop, you’d add a cancel channel here.)
}

// start parses the given CIDR (adding "/24" if missing), then spins up the background ping loop.
func (m *Manager) start(cidr string) {
	if strings.TrimSpace(cidr) == "" {
		fmt.Printf("Network scanner not started: no subnet configured.")
		return
	}

	// If the user omitted the mask (e.g. "10.0.100.0"), default to /24
	if !strings.Contains(cidr, "/") {
		cidr = cidr + "/24"
	}

	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		fmt.Printf("Network scanner not started: invalid CIDR \"%s\": %v", cidr, err)
		return
	}

	sc := &scanner{
		Subnet:       ipnet,
		Interval:     30 * time.Second,
		Devices:      make(map[string]Device),
		KnownDevices: map[string]bool{"10.0.100.5": true, "10.0.100.3": true, "10.0.100.2": true},
	}
	m.scanner = sc

	go func() {
		for {
			sc.scan()
			time.Sleep(sc.Interval)
		}
	}()
	fmt.Printf("Network scanner started on subnet %s", cidr)
}

// GetDevices returns the last‐seen devices. Returns nil if the scanner has never started.
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

// scan performs an ICMP ping sweep of the configured subnet.
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

// incrementIP iterates to the next IP in the subnet.
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] != 0 {
			break
		}
	}
}

// ping sends a single ICMP Echo Request to the given IP, timing out after 500ms.
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
	_, err = c.WriteTo(msgBytes, dst)
	if err != nil {
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
		log.Printf("Reply from %s in %v", peer.String(), time.Since(start))
		return true, nil
	}
	return false, nil
}

// Any existing entry in s.Devices[ip].Rogue will be set to false.
func (m *Manager) MarkAsKnown(ip string) {
	if m.scanner == nil {
		return
	}
	m.scanner.Mutex.Lock()
	defer m.scanner.Mutex.Unlock()

	// Add to known devices so future scans won't mark it as rogue
	m.scanner.KnownDevices[ip] = true

	// If it's already in Devices map, update its Rogue field
	if d, ok := m.scanner.Devices[ip]; ok {
		d.Rogue = false
		m.scanner.Devices[ip] = d
	}
}
