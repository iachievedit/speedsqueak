/*
Copyright 2024 iAchieved.it LLC

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH
REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF MERCHANTABILITY
AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE
OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR
PERFORMANCE OF THIS SOFTWARE.

SPDX-License-Identifier: ISC
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pebbe/zmq4"
	"go.bug.st/serial"
)

const (
	ResetReason  = "?R"
	PartNumber   = "?P"
	SerialNumber = "?N"
	MilesPerHour = "US"
	SpeedFilter  = "R>5\r"

	Reset     = "{\"Reset\" : \"Board was reset."
	USBActive = "{\"USB\" : \"USB Interface Active\""
)

func B(s string) []byte {
	return []byte(s)
}

func readPort(port serial.Port) string {
	buff := make([]byte, 128)
	n, _ := port.Read(buff)
	opsStr := strings.TrimSpace(string(buff[:n]))

	return opsStr
}

func readPortJSON(port serial.Port) string {
	buff := make([]byte, 128)
	n := 0

	buff[0] = 0

	for buff[0] != '{' {
		n, _ = port.Read(buff)
	}
	opsStr := string(buff[:n])
	return opsStr
}

var OPS243 struct {
	product string
	serial  string
	units   string
}

var partNumber struct {
	Product string `json:"Product"`
}

var serialNumber struct {
	SerialNumber string `json:"SerialNumber"`
}

var speedUnits struct {
	Units string `json:"Units"`
}

// Initialize the OPS243 sensor
//
// Initialization consists of reading the part number, serial number,
// and finally, setting the output units to miles per hour
func initOPS243(port serial.Port) {

	port.Write(B(PartNumber))

	response := readPortJSON(port)
	fmt.Print(response)
	if err := json.Unmarshal([]byte(response), &partNumber); err != nil {
		log.Fatal("Fatal:  ", err)
	}

	OPS243.product = partNumber.Product

	log.Print("Get serial number")

	port.Write(B(SerialNumber))
	response = readPortJSON(port)
	fmt.Print(response)
	if err := json.Unmarshal([]byte(response), &serialNumber); err != nil {
		log.Fatal("Fatal:  ", err)
	}

	OPS243.serial = serialNumber.SerialNumber

	// Set output units to miles per hour

	fmt.Println("Setting speed units to miles per hour")

	port.Write(B(MilesPerHour))
	response = readPortJSON(port)
	if err := json.Unmarshal([]byte(response), &speedUnits); err != nil {
		log.Fatal("Fatal:  ", err)
	}
	OPS243.units = speedUnits.Units

	port.Write(B(SpeedFilter))
	response = readPortJSON(port) // TODO:  Check for success
	fmt.Println(response)

	fmt.Printf("Product:  %s, Serial:  %s\n", OPS243.product, OPS243.serial)

	fmt.Println(OPS243.units)

}

// SpeedEvent
type SpeedEvent struct {
	Type      string  `json:"type"`
	Timestamp string  `json:"timestamp"`
	Reading   float64 `json:"reading"`
	Units     string  `json:"units"`
	UUID      string  `json:"uuid"`
}

func main() {

	publisher, err := zmq4.NewSocket(zmq4.PUB)
	if err != nil {
		log.Fatal(err)
	}
	defer publisher.Close()

	err = publisher.Bind("tcp://*:11205")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("ZeroMQ running")

	topic := "speed/events"

	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open("/dev/ttyACM0", mode)

	if err != nil {
		log.Println("Error opening serial port")
		log.Fatal(err)
	}

	log.Println("Synchronizing with OPS243 sensor")
	ready := false
	for !ready {
		reading := readPort(port)
		_, err := strconv.ParseFloat(reading, 64)
		if err == nil {
			fmt.Println("Receiving OPS243 readings, ready!")
			ready = true
		} else {
			fmt.Println("Synchronizing")
		}
	}

	initOPS243(port)

	// Get speed
	for {
		reading := readPort(port)
		speed, _ := strconv.ParseFloat(reading, 64)

		event := SpeedEvent{
			Type:      "speed",
			Timestamp: time.Now().Format(time.RFC3339),
			Reading:   speed,
			Units:     "mph",
			UUID:      uuid.NewString(),
		}

		jsonData, err := json.Marshal(event)
		if err != nil {
			log.Panic(err)
		} else {
			_, err := publisher.Send(fmt.Sprintf("%s %s", topic, string(jsonData)), 0)
			if err != nil {
				log.Panic(err)
			}
			fmt.Println("Sent:  ", string(jsonData))
		}

	}

}
