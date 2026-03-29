package serial

import (
	"log"
	"strings"
	"time"

	goserial "go.bug.st/serial"
)

// ListPorts returns all available serial ports on the system.
func ListPorts() ([]string, error) {
	return goserial.GetPortsList()
}

// ReadLoop opens the serial port and calls onTrigger for each "TRIGGER" line.
// It retries the connection on error so the server survives Arduino resets.
// getPort returns the port to use; return "" for auto-discovery.
// onConnect receives the name of the port that connected.
func ReadLoop(getPort func() string, baudRate int, onTrigger func(), onConnect func(string), onDisconnect func()) {
	for {
		port := getPort()
		if port == "" {
			port = discoverPort()
		}
		if port == "" {
			time.Sleep(2 * time.Second)
			continue
		}
		if err := readOnce(port, baudRate, onTrigger, onConnect); err != nil {
			log.Printf("serial: disconnected (%v) — retrying in 2s", err)
		}
		onDisconnect()
		time.Sleep(2 * time.Second)
	}
}

// discoverPort returns the first available port matching known Arduino patterns.
func discoverPort() string {
	ports, err := goserial.GetPortsList()
	if err != nil || len(ports) == 0 {
		return ""
	}
	patterns := []string{"usbserial", "usbmodem", "ttyUSB", "ttyACM"}
	for _, p := range ports {
		for _, pat := range patterns {
			if strings.Contains(p, pat) {
				log.Printf("serial: auto-discovered port %s", p)
				return p
			}
		}
	}
	return ""
}

func readOnce(portName string, baudRate int, onTrigger func(), onConnect func(string)) error {
	mode := &goserial.Mode{BaudRate: baudRate}
	port, err := goserial.Open(portName, mode)
	if err != nil {
		return err
	}
	defer port.Close()
	// 1-second read timeout so disconnection is detected promptly
	if err := port.SetReadTimeout(time.Second); err != nil {
		return err
	}
	log.Printf("serial: connected to %s at %d baud", portName, baudRate)
	onConnect(portName)

	buf := make([]byte, 64)
	var line strings.Builder
	for {
		n, err := port.Read(buf)
		if err != nil {
			return err // real error — device disconnected
		}
		for _, b := range buf[:n] {
			if b == '\n' {
				if strings.TrimSpace(line.String()) == "TRIGGER" {
					onTrigger()
				}
				line.Reset()
			} else {
				line.WriteByte(b)
			}
		}
	}
}
