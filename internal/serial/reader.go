package serial

import (
	"bufio"
	"log"
	"strings"
	"time"

	goserial "go.bug.st/serial"
)

// ReadLoop opens the serial port and calls onTrigger for each "TRIGGER" line.
// It retries the connection on error so the server survives Arduino resets.
func ReadLoop(portName string, baudRate int, onTrigger func()) {
	for {
		if err := readOnce(portName, baudRate, onTrigger); err != nil {
			log.Printf("serial: disconnected (%v) — retrying in 2s", err)
		}
		time.Sleep(2 * time.Second)
	}
}

func readOnce(portName string, baudRate int, onTrigger func()) error {
	mode := &goserial.Mode{BaudRate: baudRate}
	port, err := goserial.Open(portName, mode)
	if err != nil {
		return err
	}
	defer port.Close()
	log.Printf("serial: connected to %s at %d baud", portName, baudRate)

	scanner := bufio.NewScanner(port)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "TRIGGER" {
			onTrigger()
		}
	}
	return scanner.Err()
}
