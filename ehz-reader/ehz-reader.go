package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"flag"
	"log"
	"time"

	"github.com/tarm/serial"
)

type measurement struct {
	name       string
	pattern    []byte
	startIndex int
	length     int
	divisor    float64
}

var measurements = []measurement{
	measurement{name: "power", pattern: []byte{'\x07', '\x01', '\x00', '\x10', '\x07', '\x00'}, startIndex: 8, length: 4, divisor: 1},
	measurement{name: "total", pattern: []byte{'\x07', '\x01', '\x00', '\x01', '\x08', '\x00'}, startIndex: 12, length: 8, divisor: 10000}}

func splitMsg(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{'\x1b', '\x1b', '\x1b', '\x1b', '\x01', '\x01', '\x01', '\x01'}); i >= 0 {
		// We have a full newline-terminated line.
		return i + 2, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

func parseMsg(msg []byte) {
	// log.Printf("%x\n", msg)
	var fields = make(map[string]interface{})
	for _, m := range measurements {
		if i := bytes.Index(msg, m.pattern); i > 0 {
			l := len(m.pattern)
			start := i + l + m.startIndex
			slice := msg[start : start+m.length]
			var value float64
			if m.length == 8 {
				value = float64(binary.BigEndian.Uint64(slice))
			} else {
				value = float64(binary.BigEndian.Uint32(slice))
			}
			fields[m.name] = value / m.divisor
		}
	}
	log.Printf("fields: %v", fields)
	if len(fields) > 0 {
		writePoints(fields)
	}
}

var device = flag.String("device", getenv("EHZ_DEVICE", "/dev/ttyUSB0"), "usb reader device")

func main() {
	influxConfig := InfluxConfig{
		URI:         *flag.String("influx", getenv("EHZ_INFLUX_URI", "http://influxdb:8086"), "influx uri"),
		Database:    *flag.String("db", getenv("EHZ_INFLUX_DB", "home"), "database name"),
		Measurement: *flag.String("measurement", getenv("EHZ_INFLUX_MEASUREMENT", "electric_meter"), "measurement"),
		Meter:       *flag.String("meter", getenv("EHZ_INFLUX_METER", "main"), "value of meter tag"),
	}

	flag.Parse()

	setupInflux(influxConfig)

	c := &serial.Config{Name: *device, Baud: 9600, ReadTimeout: time.Second * 3}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(s)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 2048), 4*1024)
	scanner.Split(splitMsg)

	for scanner.Scan() {
		go parseMsg(scanner.Bytes())
	}
}
