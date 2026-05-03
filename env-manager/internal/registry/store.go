package registry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Store manages the service registry using JSON files (lightweight, no CGO/SQLite needed)
type Store struct {
	dataDir string
}

type Service struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Node      string    `json:"node"`
	Port      int       `json:"port"`
	Proto     string    `json:"proto"`
	Source    string    `json:"source"` // discovered | manual
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Metadata  string    `json:"metadata"` // JSON blob
}

type PortUsage struct {
	Node    string `json:"node"`
	Port    int    `json:"port"`
	Process string `json:"process"`
	InUse   bool   `json:"in_use"`
}

// registryData is the in-memory + persisted structure
type registryData struct {
	Services  []Service `json:"services"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	return &Store{dataDir: dataDir}, nil
}

// New is an alias for NewStore
func New(dataDir string) (*Store, error) {
	return NewStore(dataDir)
}

func (s *Store) dataFile() string {
	return filepath.Join(s.dataDir, "registry.json")
}

func (s *Store) loadData() (registryData, error) {
	var data registryData
	data.Services = []Service{}

	content, err := os.ReadFile(s.dataFile())
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return data, err
	}

	if err := json.Unmarshal(content, &data); err != nil {
		return data, fmt.Errorf("parse registry: %w", err)
	}
	return data, nil
}

func (s *Store) saveData(data registryData) error {
	data.UpdatedAt = time.Now()
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.dataFile(), content, 0644)
}

func (s *Store) RegisterService(svc Service) error {
	data, err := s.loadData()
	if err != nil {
		return err
	}

	// Update existing or append new
	found := false
	for i := range data.Services {
		if data.Services[i].Node == svc.Node && data.Services[i].Port == svc.Port && data.Services[i].Proto == svc.Proto {
			data.Services[i].Name = svc.Name
			data.Services[i].LastSeen = svc.LastSeen
			data.Services[i].Metadata = svc.Metadata
			if svc.Source != "" {
				data.Services[i].Source = svc.Source
			}
			found = true
			break
		}
	}

	if !found {
		// Assign ID
		maxID := int64(0)
		for _, existing := range data.Services {
			if existing.ID > maxID {
				maxID = existing.ID
			}
		}
		svc.ID = maxID + 1
		if svc.FirstSeen.IsZero() {
			svc.FirstSeen = time.Now()
		}
		if svc.LastSeen.IsZero() {
			svc.LastSeen = time.Now()
		}
		data.Services = append(data.Services, svc)
	}

	return s.saveData(data)
}

func (s *Store) ListServices(node string, port int, name string) ([]Service, error) {
	data, err := s.loadData()
	if err != nil {
		return nil, err
	}

	var results []Service
	for _, svc := range data.Services {
		if node != "" && svc.Node != node {
			continue
		}
		if port > 0 && svc.Port != port {
			continue
		}
		if name != "" && !strings.Contains(strings.ToLower(svc.Name), strings.ToLower(name)) {
			continue
		}
		results = append(results, svc)
	}

	// Sort by last_seen desc
	sort.Slice(results, func(i, j int) bool {
		return results[i].LastSeen.After(results[j].LastSeen)
	})

	return results, nil
}

func (s *Store) FindPort(port int, node string) (*PortUsage, error) {
	data, err := s.loadData()
	if err != nil {
		return nil, err
	}

	for _, svc := range data.Services {
		if svc.Port != port {
			continue
		}
		if node != "" && svc.Node != node {
			continue
		}
		return &PortUsage{
			Node:    svc.Node,
			Port:    svc.Port,
			Process: svc.Name,
			InUse:   true,
		}, nil
	}

	return &PortUsage{Port: port, InUse: false}, nil
}

func (s *Store) SaveSnapshot(node string, data string) error {
	snapFile := filepath.Join(s.dataDir, fmt.Sprintf("snapshot-%s-%s.json", node, time.Now().Format("20060102-150405")))
	return os.WriteFile(snapFile, []byte(data), 0644)
}

func (s *Store) Close() error {
	return nil // No resources to close for file-based storage
}
