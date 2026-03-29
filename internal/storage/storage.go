package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type SessionRecord struct {
	ID        string    `json:"id"`        // filename stem, used as key
	StartedAt time.Time `json:"startedAt"`
	StoppedAt time.Time `json:"stoppedAt"`
	TotalTime float64   `json:"totalTime"`
	LapCount  int       `json:"lapCount"`
	BestLap   float64   `json:"bestLap"`
	Laps      []float64 `json:"laps"`
}

const dir = "./sessions"

func Save(rec SessionRecord) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	rec.ID = rec.StartedAt.Format("2006-01-02T15-04-05")
	path := filepath.Join(dir, rec.ID+".json")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(rec)
}

func LoadAll() ([]SessionRecord, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var records []SessionRecord
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var rec SessionRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			continue
		}
		records = append(records, rec)
	}

	// Newest first
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartedAt.After(records[j].StartedAt)
	})
	return records, nil
}
