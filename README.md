# Mini 4WD Lap Timer

A real-time lap timer for Mini 4WD racing, built with Go and vanilla JavaScript. An Arduino with an IR sensor detects each car pass and triggers lap recording via serial port. Results are broadcast live to the browser over WebSocket.

## Features

- Real-time current lap and total race time display
- Lap history with best lap highlighted
- Pause, resume, and reset support
- Session history saved to disk
- Race sounds (countdown, GO, lap bell, best lap arpeggio)

## Requirements

- Go 1.22+
- Arduino with IR sensor connected via USB serial
- A browser

## Hardware Setup

1. Connect the IR receiver output to Arduino **D2**
2. Flash the sketch at `arduino/lap_sensor/lap_sensor.ino`
3. Connect the Arduino to the computer via USB

The Arduino sends a `TRIGGER` line over serial (9600 baud) each time the IR beam is broken, with a 500 ms debounce.

## Running

```bash
go run ./cmd/server
```

Then open `http://localhost:8080` in a browser.

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | `/dev/ttyUSB0` | Serial port the Arduino is connected to |
| `-addr` | `:8080` | HTTP listen address |

Example:

```bash
go run ./cmd/server -port /dev/tty.usbmodem101 -addr :9000
```

## Session Data

Completed sessions are saved as JSON files in the `sessions/` directory. The in-app session history panel loads them on demand.

## Project Structure

```
cmd/server/        - HTTP server, WebSocket, REST endpoints
internal/
  session/         - Lap timing state machine
  hub/             - WebSocket broadcast hub
  serial/          - Serial port listener
  storage/         - Session persistence
static/            - Frontend (HTML, CSS, JS)
arduino/           - Arduino IR sensor sketch
sessions/          - Saved session JSON files (git-ignored)
```
