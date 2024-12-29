// Copyright 20## Team ###. All Rights Reserved.
// Author: cpapplefamily@gmail.com (Corey Applegate)
//
// Alternate IO handlers for the ###.

package plc

import (
	//"github.com/Team254/cheesy-arena/game"
	//"github.com/Team254/cheesy-arena/model"
	//"encoding/json"
	//"net/http"
	"time"
	"log"
	"net"
	"strings"
)

type Esp32 interface {
	Run()
	IsScoreHealthy() bool
	SetAddress(string) 
}

type Esp32IO struct {
	ScoreTableIP		string
	scoreTableHealthy 	bool
}
const LoopPeriodMs = 1000 // Define the loop period in milliseconds


// RequestPayload represents the structure of the incoming POST data.
type RequestPayload struct {
	Channel int  `json:"channel"`
	State   bool `json:"state"`
}

func (esp32 *Esp32IO) SetAddress(address string) {
	esp32.ScoreTableIP = strings.TrimSpace(address)
	log.Printf("Set ScoreTableIP to: %s", esp32.ScoreTableIP)
	/* plc.resetConnection()

	if plc.ioChangeNotifier == nil {
		// Register a notifier that listeners can subscribe to to get websocket updates about I/O value changes.
		plc.ioChangeNotifier = websocket.NewNotifier("plcIoChange", plc.generateIoChangeMessage)
	} */
}

// Checks if an IP address is reachable by attempting a TCP connection.
func isDevicePresent(ip string, port string) bool {
    address := net.JoinHostPort(ip, port)
    conn, err := net.DialTimeout("tcp", address, time.Second*2)
    if err != nil {
        log.Printf("Device not reachable at %s: %v", address, err)
        return false
    }
    conn.Close()
    return true
}

// Loops indefinitely to read inputs from and write outputs to PLC.
func (esp32 *Esp32IO) Run() {
	for {
		if !true {
			// No PLC is configured; just allow the loop to continue to simulate inputs and outputs.
			esp32.scoreTableHealthy = false
		} else {
			if !isDevicePresent(esp32.ScoreTableIP, "80")  {
				time.Sleep(time.Second * plcRetryIntevalSec)
				esp32.scoreTableHealthy = false
				continue
			}else{
				esp32.scoreTableHealthy = true
			}
		}
		log.Println("ScoreTable Check")
		startTime := time.Now()
		time.Sleep(time.Until(startTime.Add(time.Millisecond * LoopPeriodMs)))
	}
}

// Returns the health status of the alternate IO.
func (esp32 *Esp32IO) IsScoreHealthy() bool {
	return esp32.scoreTableHealthy
}
