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
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pebbe/zmq4"
	"github.com/rs/zerolog"
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

var logger zerolog.Logger

func B(s string) []byte {
	return []byte(s)
}

func readPort(port serial.Port) string {
	buff := make([]byte, 128)
	n, _ := port.Read(buff)
	opsStr := strings.TrimSpace(string(buff[:n]))

	readingsArray := strings.Split(opsStr, "\r\n")
	if len(readingsArray) > 0 {
		return readingsArray[len(readingsArray)-1]
	} else {
		return ""
	}

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
	logger.Info().Msg(response)
	if err := json.Unmarshal([]byte(response), &partNumber); err != nil {
		log.Fatal("Fatal:  ", err)
	}

	OPS243.product = partNumber.Product

	//log.Print("Get serial number")

	port.Write(B(SerialNumber))
	response = readPortJSON(port)
	//fmt.Print(response)
	logger.Info().Msg(response)
	if err := json.Unmarshal([]byte(response), &serialNumber); err != nil {
		log.Fatal("Fatal:  ", err)
	}

	OPS243.serial = serialNumber.SerialNumber

	// Set output units to miles per hour

	logger.Info().Msg("Setting speed units to miles per hour")

	port.Write(B(MilesPerHour))
	response = readPortJSON(port)
	if err := json.Unmarshal([]byte(response), &speedUnits); err != nil {
		log.Fatal("Fatal:  ", err)
	}
	OPS243.units = speedUnits.Units

	port.Write(B(SpeedFilter))
	response = readPortJSON(port) // TODO:  Check for success
	//fmt.Println(response)

	//fmt.Printf("Product:  %s, Serial:  %s\n", OPS243.product, OPS243.serial)

	//fmt.Println(OPS243.units)

}

// SpeedEvent
type SpeedEvent struct {
	Type      string  `json:"type"`
	Timestamp string  `json:"timestamp"`
	Reading   float64 `json:"reading"`
	Direction string  `json:"direction"`
	Units     string  `json:"units"`
	UUID      string  `json:"uuid"`
	Raw       string  `json:"raw"`
}

func main() {

	logFilePath := "/var/log/speedsqueak/ops243.log"

	// Open the log file for writing
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	// Create a zerolog logger with file output
	logger = zerolog.New(logFile).With().
		Timestamp().
		Str("service", "speedsqueak-radar").
		Logger()

	publisher, err := zmq4.NewSocket(zmq4.PUB)
	if err != nil {
		log.Fatal(err)
	}
	defer publisher.Close()

	err = publisher.Bind("tcp://*:11205")
	if err != nil {
		log.Fatal(err)
	}

	logger.Info().Msg("ZeroMQ running")

	topic := "event/speed"

	mode := &serial.Mode{
		BaudRate: 115200,
	}

	port, err := serial.Open("/dev/ttyACM0", mode)

	if err != nil {
		log.Println("Error opening serial port")
		log.Fatal(err)
	}

	initOPS243(port)

	lastEvent := time.Now().Unix()

	// Get speeding events
	for {
		reading := readPort(port)
		speed, _ := strconv.ParseFloat(reading, 64)

		direction := "toward"

		if speed < 0 {
			direction = "away"
			speed = math.Abs(speed)
		}

		now := time.Now().Unix()

		if now-lastEvent < 5 {
			continue
		}

		event := SpeedEvent{
			Type:      "speed",
			Timestamp: time.Now().Format(time.RFC3339),
			Reading:   speed,
			Direction: direction,
			Units:     "mph",
			UUID:      uuid.NewString(),
			Raw:       reading,
		}

		jsonData, err := json.Marshal(event)
		if err != nil {
			log.Panic(err)
		} else {
			_, err := publisher.Send(fmt.Sprintf("%s %s", topic, string(jsonData)), 0)
			lastEvent = now
			if err != nil {
				log.Panic(err)
			}
			logger.Info().Interface("event", event).Msg("Reading sent")
		}

	}

}
