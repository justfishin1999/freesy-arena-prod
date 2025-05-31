// network/switch.go
package network

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Team254/cheesy-arena/model"
)

const (
	switchConfigBackoffDurationSec = 5
	switchConfigPauseDurationSec   = 2
	switchTeamGatewayAddress       = 4
	switchTelnetPort               = 23
	ServerIpAddress                = "10.0.100.5"
)

const (
	red1Vlan  = 10
	red2Vlan  = 20
	red3Vlan  = 30
	blue1Vlan = 40
	blue2Vlan = 50
	blue3Vlan = 60
)

type Switch struct {
	vendor                string
	address               string
	port                  int
	password              string
	mutex                 sync.Mutex
	configBackoffDuration time.Duration
	configPauseDuration   time.Duration
	Status                string
}

func NewCisco3000Switch(address, password string) *Switch {
	return &Switch{
		vendor:                "Cisco",
		address:               address,
		port:                  switchTelnetPort,
		password:              password,
		configBackoffDuration: switchConfigBackoffDurationSec * time.Second,
		configPauseDuration:   switchConfigPauseDurationSec * time.Second,
		Status:                "UNKNOWN",
	}
}

func NewArubaSwitch(address, password string) *Switch {
	return &Switch{
		vendor:                "Aruba",
		address:               address,
		port:                  switchTelnetPort,
		password:              password,
		configBackoffDuration: switchConfigBackoffDurationSec * time.Second,
		configPauseDuration:   switchConfigPauseDurationSec * time.Second,
		Status:                "UNKNOWN",
	}
}

func NewCiscoISR(address, password string) *Switch {
	return &Switch{
		vendor:                "Cisco ISR",
		address:               address,
		port:                  switchTelnetPort,
		password:              password,
		configBackoffDuration: switchConfigBackoffDurationSec * time.Second,
		configPauseDuration:   switchConfigPauseDurationSec * time.Second,
		Status:                "UNKNOWN",
	}
}

func (sw *Switch) ConfigureCisco3000Teams(teams [6]*model.Team) error {
	// Make sure multiple configurations aren't being set at the same time.
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	sw.Status = "CONFIGURING"

	// Remove old team VLANs to reset the switch state.
	removeTeamVlansCommand := ""
	for vlan := 10; vlan <= 60; vlan += 10 {
		removeTeamVlansCommand += fmt.Sprintf(
			"interface Vlan%d\nno ip address\nno ip dhcp pool dhcp%d\n", vlan, vlan,
		)
	}
	_, err := sw.runConfigCommand(removeTeamVlansCommand)
	if err != nil {
		sw.Status = "ERROR"
		return err
	}
	time.Sleep(sw.configPauseDuration)

	// Create the new team VLANs.
	addTeamVlansCommand := ""
	addTeamVlan := func(team *model.Team, vlan int) {
		if team == nil {
			return
		}
		teamPartialIp := fmt.Sprintf("%d.%d", team.Id/100, team.Id%100)
		addTeamVlansCommand += fmt.Sprintf(
			"ip dhcp excluded-address 10.%s.1 10.%s.19\n"+
				"ip dhcp excluded-address 10.%s.200 10.%s.254\n"+
				"ip dhcp pool dhcp%d\n"+
				"network 10.%s.0 255.255.255.0\n"+
				"default-router 10.%s.%d\n"+
				"lease 1\n"+
				"interface Vlan%d\nip address 10.%s.%d 255.255.255.0\n",
			teamPartialIp,
			teamPartialIp,
			teamPartialIp,
			teamPartialIp,
			vlan,
			teamPartialIp,
			teamPartialIp,
			switchTeamGatewayAddress,
			vlan,
			teamPartialIp,
			switchTeamGatewayAddress,
		)
	}
	addTeamVlan(teams[0], red1Vlan)
	addTeamVlan(teams[1], red2Vlan)
	addTeamVlan(teams[2], red3Vlan)
	addTeamVlan(teams[3], blue1Vlan)
	addTeamVlan(teams[4], blue2Vlan)
	addTeamVlan(teams[5], blue3Vlan)
	if len(addTeamVlansCommand) > 0 {
		_, err = sw.runConfigCommand(addTeamVlansCommand)
		if err != nil {
			sw.Status = "ERROR"
			return err
		}
	}

	// Give some time for the configuration to take before another one can be attempted.
	time.Sleep(sw.configBackoffDuration)

	sw.Status = "ACTIVE"
	return nil
}

func (sw *Switch) ConfigureArubaTeams(teams [6]*model.Team) error {
	// Make sure multiple configurations aren't being set at the same time.
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	sw.Status = "CONFIGURING"

	// Remove old VLAN configurations to reset the switch state.
	removeTeamVlansCommand := "configure terminal\n"
	for vlan := 10; vlan <= 60; vlan += 10 {
		removeTeamVlansCommand += fmt.Sprintf(
			"no vlan %d\n"+
				"no dhcp-server pool dhcp%d\n",
			vlan, vlan,
		)
	}
	_, err := sw.runArubaConfigCommand(removeTeamVlansCommand)
	if err != nil {
		sw.Status = "ERROR"
		return err
	}
	time.Sleep(sw.configPauseDuration)

	// Create new VLANs and configure interfaces and DHCP.
	addTeamVlansCommand := "configure terminal\n"
	addTeamVlan := func(team *model.Team, vlan int) {
		if team == nil {
			return
		}
		teamPartialIp := fmt.Sprintf("%d.%d", team.Id/100, team.Id%100)
		addTeamVlansCommand += fmt.Sprintf(
			"vlan %d\n"+
				"name TEAM_%d\n"+
				"interface vlan %d\n"+
				"ip address 10.%s.%d 255.255.255.0\n"+
				"dhcp-server\n"+
				"exit\n"+
				"dhcp-server pool dhcp%d\n"+
				"network 10.%s.0 255.255.255.0\n"+
				"gateway 10.%s.%d\n"+
				"lease-time 86400\n"+
				"excluded-address 10.%s.1 10.%s.19\n"+
				"excluded-address 10.%s.200 10.%s.254\n",
			vlan,
			team.Id,
			vlan,
			teamPartialIp,
			switchTeamGatewayAddress,
			vlan,
			teamPartialIp,
			teamPartialIp,
			switchTeamGatewayAddress,
			teamPartialIp,
			teamPartialIp,
			teamPartialIp,
			teamPartialIp,
		)
	}
	addTeamVlan(teams[0], red1Vlan)
	addTeamVlan(teams[1], red2Vlan)
	addTeamVlan(teams[2], red3Vlan)
	addTeamVlan(teams[3], blue1Vlan)
	addTeamVlan(teams[4], blue2Vlan)
	addTeamVlan(teams[5], blue3Vlan)

	if len(addTeamVlansCommand) > 0 {
		_, err = sw.runArubaConfigCommand(addTeamVlansCommand)
		if err != nil {
			sw.Status = "ERROR"
			return err
		}
	}

	// Give some time for the configuration to take before another one can be attempted.
	time.Sleep(sw.configBackoffDuration)

	sw.Status = "ACTIVE"
	return nil
}

func (sw *Switch) ConfigureCiscoISRTeams(teams [6]*model.Team) error {
	// Make sure multiple configurations aren't being set at the same time.
	log.Printf("Configuring cisco ISR")
	sw.mutex.Lock()
	defer sw.mutex.Unlock()
	sw.Status = "CONFIGURING"

	// Remove old team VLANs to reset the switch state.
	removeTeamVlansCommand := ""
	for vlan := 10; vlan <= 60; vlan += 10 {
		removeTeamVlansCommand += fmt.Sprintf(
			"interface GigabitEthernet 0/0.%d\nno ip address\nno access-list 1%d\nno ip dhcp pool dhcp%d\n", vlan, vlan, vlan,
		)
	}
	_, err := sw.runConfigCommandCiscoISR(removeTeamVlansCommand)
	if err != nil {
		sw.Status = "ERROR"
		return err
	}
	time.Sleep(sw.configPauseDuration)

	// Create the new team VLANs.
	addTeamVlansCommand := ""
	addTeamVlan := func(team *model.Team, vlan int) {
		if team == nil {
			return
		}
		teamPartialIp := fmt.Sprintf("%d.%d", team.Id/100, team.Id%100)
		addTeamVlansCommand += fmt.Sprintf(
			"ip dhcp excluded-address 10.%s.1 10.%s.19\n"+
				"ip dhcp excluded-address 10.%s.200 10.%s.254\n"+
				"ip dhcp pool dhcp%d\n"+
				"network 10.%s.0 255.255.255.0\n"+
				"default-router 10.%s.%d\n"+
				"lease 7\n"+
				"access-list 1%d permit ip 10.%s.0 0.0.0.255 host %s\n"+
				"access-list 1%d permit udp any eq bootpc any eq bootps\n"+
				"access-list 1%d permit icmp any any\n"+
				"interface GigabitEthernet 0/0.%d\nip address 10.%s.%d 255.255.255.0\n",
			teamPartialIp,
			teamPartialIp,
			teamPartialIp,
			teamPartialIp,
			vlan,
			teamPartialIp,
			teamPartialIp,
			switchTeamGatewayAddress,
			vlan,
			teamPartialIp,
			ServerIpAddress,
			vlan,
			vlan,
			vlan,
			teamPartialIp,
			switchTeamGatewayAddress,
		)
	}
	addTeamVlan(teams[0], red1Vlan)
	addTeamVlan(teams[1], red2Vlan)
	addTeamVlan(teams[2], red3Vlan)
	addTeamVlan(teams[3], blue1Vlan)
	addTeamVlan(teams[4], blue2Vlan)
	addTeamVlan(teams[5], blue3Vlan)
	if len(addTeamVlansCommand) > 0 {
		_, err = sw.runConfigCommandCiscoISR(addTeamVlansCommand)
		if err != nil {
			sw.Status = "ERROR"
			return err
		}
	}

	// Give some time for the configuration to take before another one can be attempted.
	time.Sleep(sw.configBackoffDuration)

	sw.Status = "ACTIVE"
	return nil
}

// Handle sending config commands to Cisco ISR (Router)
func (sw *Switch) runCommandCiscoISR(command string) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sw.address, sw.port))
	if err != nil {
		return "", fmt.Errorf("failed to connect: %v", err)
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	// Read initial prompt (e.g., "Password:")
	if err != nil {
		return "", fmt.Errorf("failed to read prompt: %v", err)
	}

	// Send password and subsequent commands
	commands := []string{
		sw.password,
		"enable",
		sw.password,
		"terminal length 0",
		command,
		"exit",
	}
	for _, cmd := range commands {
		_, err = writer.WriteString(cmd + "\n")
		if err != nil {
			return "", fmt.Errorf("failed to send command %s: %v", cmd, err)
		}
		writer.Flush()
		time.Sleep(100 * time.Millisecond)
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(reader); err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}
	return buf.String(), nil
}

func (sw *Switch) runConfigCommandCiscoISR(command string) (string, error) {
	output, err := sw.runCommandCiscoISR(fmt.Sprintf("config terminal\n%send\ncopy running-config startup-config\n\n", command))
	if err != nil {
		return output, err
	}
	if strings.Contains(output, "% Invalid input") || strings.Contains(output, "% Incomplete command") {
		return output, fmt.Errorf("command execution failed: %s", output)
	}
	return output, nil
}

// Handle sending config commands to default (Cisco) switch
func (sw *Switch) runCommand(command string) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sw.address, sw.port))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("%s\nenable\n%s\nterminal length 0\n%sexit\n",
		sw.password, sw.password, command))
	if err != nil {
		return "", err
	}
	if err = writer.Flush(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(conn); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (sw *Switch) runConfigCommand(command string) (string, error) {
	return sw.runCommand(fmt.Sprintf("config terminal\n%send\ncopy running-config startup-config\n\n", command))
}

// Handle sending configuration to Aruba switches
func (sw *Switch) runArubaCommand(command string) (string, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", sw.address, sw.port))
	if err != nil {
		return "", err
	}
	defer conn.Close()

	writer := bufio.NewWriter(conn)
	_, err = writer.WriteString(fmt.Sprintf("%s\n%s\nconfigure terminal\n%s\nterminal length 0\n%s\n",
		sw.password, sw.password, command, sw.password))
	if err != nil {
		return "", err
	}
	if err = writer.Flush(); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if _, err = buf.ReadFrom(conn); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (sw *Switch) runArubaConfigCommand(command string) (string, error) {
	// Ensure we are in configuration mode and perform the operation
	return sw.runArubaCommand(fmt.Sprintf("configure terminal\n%s\nwrite memory\n", command)) // 'write memory' to save config
}
