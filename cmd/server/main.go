package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"

	"laptimer/internal/hub"
	"laptimer/internal/serial"
	"laptimer/internal/session"
	"laptimer/internal/storage"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func main() {
	portFlag := flag.String("port", "", "serial port (leave empty for auto-discovery)")
	addrFlag := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	sess := session.New()
	h := hub.New()
	go h.Run()

	// --- Serial port state ---
	var (
		serialConnected atomic.Bool
		currentPortMu   sync.RWMutex
		currentPort     string // connected port name (empty when disconnected)
		selectedPort    = *portFlag
		selectedPortMu  sync.RWMutex
	)

	getSelectedPort := func() string {
		selectedPortMu.RLock()
		defer selectedPortMu.RUnlock()
		return selectedPort
	}

	serialStatus := func() map[string]interface{} {
		currentPortMu.RLock()
		defer currentPortMu.RUnlock()
		return map[string]interface{}{
			"serialConnected": serialConnected.Load(),
			"portName":        currentPort,
		}
	}

	broadcastSerial := func(connected bool, portName string) {
		serialConnected.Store(connected)
		currentPortMu.Lock()
		currentPort = portName
		currentPortMu.Unlock()
		data, _ := json.Marshal(serialStatus())
		h.Broadcast(data)
	}

	// Serial -> Session -> Broadcast
	go serial.ReadLoop(getSelectedPort, 9600,
		func() {
			state := sess.RecordTrigger()
			data, _ := json.Marshal(state)
			h.Broadcast(data)
		},
		func(portName string) { broadcastSerial(true, portName) },
		func() { broadcastSerial(false, "") },
	)

	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("ws upgrade error: %v", err)
			return
		}
		h.Register(conn)

		state := sess.CurrentState()
		if data, err := json.Marshal(state); err == nil {
			conn.WriteMessage(websocket.TextMessage, data)
		}
		if data, err := json.Marshal(serialStatus()); err == nil {
			conn.WriteMessage(websocket.TextMessage, data)
		}

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				h.Unregister(conn)
				return
			}
		}
	})

	postHandler := func(action func() session.State) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			state := action()
			data, _ := json.Marshal(state)
			h.Broadcast(data)
			w.WriteHeader(http.StatusOK)
		}
	}

	http.HandleFunc("/arm", postHandler(sess.Arm))
	http.HandleFunc("/pause", postHandler(sess.Pause))
	http.HandleFunc("/reset-time", postHandler(sess.ResetTime))

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		state := sess.Stop()
		saveSession(sess)
		data, _ := json.Marshal(state)
		h.Broadcast(data)
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		saveSession(sess)
		sess.Reset()
		state := sess.CurrentState()
		data, _ := json.Marshal(state)
		h.Broadcast(data)
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/history", func(w http.ResponseWriter, r *http.Request) {
		records, err := storage.LoadAll()
		if err != nil {
			http.Error(w, "failed to load history", http.StatusInternalServerError)
			return
		}
		if records == nil {
			records = []storage.SessionRecord{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
	})

	http.HandleFunc("/ports", func(w http.ResponseWriter, r *http.Request) {
		ports, err := serial.ListPorts()
		if err != nil {
			http.Error(w, "failed to list ports", http.StatusInternalServerError)
			return
		}
		if ports == nil {
			ports = []string{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ports)
	})

	http.HandleFunc("/port", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Port string `json:"port"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		selectedPortMu.Lock()
		selectedPort = body.Port
		selectedPortMu.Unlock()
		log.Printf("serial: port changed to %q", body.Port)
		w.WriteHeader(http.StatusOK)
	})

	log.Printf("Lap timer server listening on %s", *addrFlag)
	log.Printf("Serial port: %s", *portFlag)
	log.Fatal(http.ListenAndServe(*addrFlag, nil))
}

func saveSession(sess *session.Session) {
	startedAt, stoppedAt, laps, ok := sess.Snapshot()
	if !ok {
		return
	}
	rec := storage.SessionRecord{
		StartedAt: startedAt,
		StoppedAt: stoppedAt,
		LapCount:  len(laps),
		Laps:      make([]float64, len(laps)),
	}
	var best float64
	for i, d := range laps {
		s := d.Seconds()
		rec.Laps[i] = s
		rec.TotalTime += s
		if best == 0 || s < best {
			best = s
		}
	}
	rec.BestLap = best
	if err := storage.Save(rec); err != nil {
		log.Printf("failed to save session: %v", err)
	}
}
