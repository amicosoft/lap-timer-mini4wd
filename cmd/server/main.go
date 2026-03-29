package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

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
	portFlag := flag.String("port", "/dev/ttyUSB0", "serial port (e.g. /dev/tty.usbmodemXXXX or COM3)")
	addrFlag := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	sess := session.New()
	h := hub.New()
	go h.Run()

	// Serial -> Session -> Broadcast
	go serial.ReadLoop(*portFlag, 9600, func() {
		state := sess.RecordTrigger()
		data, _ := json.Marshal(state)
		h.Broadcast(data)
	})

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
