// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package network

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/Team254/cheesy-arena/model"
	"github.com/stretchr/testify/assert"
)

func TestConfigureSwitch(t *testing.T) {
	sw := NewCisco3000Switch("127.0.0.1", "password")
	assert.Equal(t, "UNKNOWN", sw.Status)
	sw.port = 9050
	sw.configBackoffDuration = time.Millisecond
	sw.configPauseDuration = time.Millisecond
	var command1, command2 string

	expectedResetCommand := "password\nenable\npassword\nterminal length 0\nconfig terminal\n" +
		"interface Vlan10\nno ip address\nno ip dhcp pool dhcp10\n" +
		"interface Vlan20\nno ip address\nno ip dhcp pool dhcp20\n" +
		"interface Vlan30\nno ip address\nno ip dhcp pool dhcp30\n" +
		"interface Vlan40\nno ip address\nno ip dhcp pool dhcp40\n" +
		"interface Vlan50\nno ip address\nno ip dhcp pool dhcp50\n" +
		"interface Vlan60\nno ip address\nno ip dhcp pool dhcp60\n" +
		"end\n" +
		"copy running-config startup-config\n\n" +
		"exit\n"

	expectedOneTeam :=
		"password\nenable\npassword\nterminal length 0\nconfig terminal\n" +
			"ip dhcp excluded-address 10.2.54.1 10.2.54.19\n" +
			"ip dhcp excluded-address 10.2.54.200 10.2.54.254\n" +
			"ip dhcp pool dhcp50\n" +
			"network 10.2.54.0 255.255.255.0\n" +
			"default-router 10.2.54.4\n" +
			"lease 1\n" +
			"interface Vlan50\n" +
			"ip address 10.2.54.4 255.255.255.0\n" +
			"end\n" +
			"copy running-config startup-config\n\n" +
			"exit\n"

	expectedAllTeams :=
		"password\nenable\npassword\nterminal length 0\nconfig terminal\n" +
			"ip dhcp excluded-address 10.11.14.1 10.11.14.19\n" +
			"ip dhcp excluded-address 10.11.14.200 10.11.14.254\n" +
			"ip dhcp pool dhcp10\n" +
			"network 10.11.14.0 255.255.255.0\n" +
			"default-router 10.11.14.4\n" +
			"lease 1\n" +
			"interface Vlan10\n" +
			"ip address 10.11.14.4 255.255.255.0\n" +
			"ip dhcp excluded-address 10.2.54.1 10.2.54.19\n" +
			"ip dhcp excluded-address 10.2.54.200 10.2.54.254\n" +
			"ip dhcp pool dhcp20\n" +
			"network 10.2.54.0 255.255.255.0\n" +
			"default-router 10.2.54.4\n" +
			"lease 1\n" +
			"interface Vlan20\n" +
			"ip address 10.2.54.4 255.255.255.0\n" +
			"ip dhcp excluded-address 10.2.96.1 10.2.96.19\n" +
			"ip dhcp excluded-address 10.2.96.200 10.2.96.254\n" +
			"ip dhcp pool dhcp30\n" +
			"network 10.2.96.0 255.255.255.0\n" +
			"default-router 10.2.96.4\n" +
			"lease 1\n" +
			"interface Vlan30\n" +
			"ip address 10.2.96.4 255.255.255.0\n" +
			"ip dhcp excluded-address 10.15.3.1 10.15.3.19\n" +
			"ip dhcp excluded-address 10.15.3.200 10.15.3.254\n" +
			"ip dhcp pool dhcp40\n" +
			"network 10.15.3.0 255.255.255.0\n" +
			"default-router 10.15.3.4\n" +
			"lease 1\n" +
			"interface Vlan40\n" +
			"ip address 10.15.3.4 255.255.255.0\n" +
			"ip dhcp excluded-address 10.16.78.1 10.16.78.19\n" +
			"ip dhcp excluded-address 10.16.78.200 10.16.78.254\n" +
			"ip dhcp pool dhcp50\n" +
			"network 10.16.78.0 255.255.255.0\n" +
			"default-router 10.16.78.4\n" +
			"lease 1\n" +
			"interface Vlan50\n" +
			"ip address 10.16.78.4 255.255.255.0\n" +
			"ip dhcp excluded-address 10.15.38.1 10.15.38.19\n" +
			"ip dhcp excluded-address 10.15.38.200 10.15.38.254\n" +
			"ip dhcp pool dhcp60\n" +
			"network 10.15.38.0 255.255.255.0\n" +
			"default-router 10.15.38.4\n" +
			"lease 1\n" +
			"interface Vlan60\n" +
			"ip address 10.15.38.4 255.255.255.0\n" +
			"end\n" +
			"copy running-config startup-config\n\n" +
			"exit\n"

	// Should remove all previous VLANs and do nothing else if current configuration is blank.
	mockTelnet(t, sw.port, &command1, &command2)
	assert.Nil(t, sw.ConfigureCisco3000Teams([6]*model.Team{nil, nil, nil, nil, nil, nil}))
	assert.Equal(t, expectedResetCommand, command1)
	assert.Equal(t, "", command2)
	assert.Equal(t, "ACTIVE", sw.Status)

	// Should configure one team if only one is present.
	sw.port += 1
	mockTelnet(t, sw.port, &command1, &command2)
	assert.Nil(t, sw.ConfigureCisco3000Teams([6]*model.Team{nil, nil, nil, nil, {Id: 254}, nil}))
	assert.Equal(t, expectedResetCommand, command1)
	assert.Equal(t, expectedOneTeam, command2)

	// Should configure all teams if all are present.
	sw.port += 1
	mockTelnet(t, sw.port, &command1, &command2)
	assert.Nil(
		t,
		sw.ConfigureCisco3000Teams([6]*model.Team{{Id: 1114}, {Id: 254}, {Id: 296}, {Id: 1503}, {Id: 1678}, {Id: 1538}}),
	)
	assert.Equal(t, expectedResetCommand, command1)
	assert.Equal(t, expectedAllTeams, command2)
}

func mockTelnet(t *testing.T, port int, command1 *string, command2 *string) {
	go func() {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		assert.Nil(t, err)
		defer ln.Close()
		*command1 = ""
		*command2 = ""

		// Fake the first connection.
		conn1, err := ln.Accept()
		assert.Nil(t, err)
		conn1.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		var reader bytes.Buffer
		reader.ReadFrom(conn1)
		*command1 = reader.String()
		conn1.Close()

		// Fake the second connection.z
		conn2, err := ln.Accept()
		assert.Nil(t, err)
		conn2.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
		reader.Reset()
		reader.ReadFrom(conn2)
		*command2 = reader.String()
		conn2.Close()
	}()
	time.Sleep(100 * time.Millisecond) // Give it some time to open the socket.
}
