package registry

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages the SQLite database for service registry and snapshots
type Store struct {
	db *sql.DB
}

type Service struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Node     string `json:"node"`
	Port     int    `json:"port"`
	Proto    string `json:"proto"`
	Source   string `json:"source"` // discovered | manual
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

func NewStore(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, "registry.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

// New is an alias for NewStore for backward compatibility
func New(dataDir string) (*Store, error) {
	return NewStore(dataDir)
}

func (s *Store) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS services (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    node        TEXT NOT NULL,
    port        INTEGER NOT NULL,
    proto       TEXT DEFAULT 'tcp',
    source      TEXT DEFAULT 'discovered',
    first_seen  DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen   DATETIME DEFAULT CURRENT_TIMESTAMP,
    metadata    TEXT,
    UNIQUE(node, port, proto)
);

CREATE TABLE IF NOT EXISTS snapshots (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    node        TEXT NOT NULL,
    timestamp   DATETIME DEFAULT CURRENT_TIMESTAMP,
    data        TEXT NOT NULL,
    type        TEXT DEFAULT 'full'
);

CREATE INDEX IF NOT EXISTS idx_services_node ON services(node);
CREATE INDEX IF NOT EXISTS idx_services_port ON services(port);
CREATE INDEX IF NOT EXISTS idx_snapshots_node_time ON snapshots(node, timestamp);
`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) RegisterService(svc Service) error {
	_, err := s.db.Exec(`
		INSERT INTO services (name, node, port, proto, source, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(node, port, proto) DO UPDATE SET
			name = excluded.name,
			last_seen = CURRENT_TIMESTAMP,
			metadata = excluded.metadata
	`, svc.Name, svc.Node, svc.Port, svc.Proto, svc.Source, svc.Metadata)
	return err
}

func (s *Store) ListServices(node string, port int, name string) ([]Service, error) {
	query := "SELECT id, name, node, port, proto, source, first_seen, last_seen, metadata FROM services WHERE 1=1"
	var args []interface{}

	if node != "" {
		query += " AND node = ?"
		args = append(args, node)
	}
	if port > 0 {
		query += " AND port = ?"
		args = append(args, port)
	}
	if name != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+name+"%")
	}

	query += " ORDER BY last_seen DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []Service
	for rows.Next() {
		var svc Service
		err := rows.Scan(&svc.ID, &svc.Name, &svc.Node, &svc.Port, &svc.Proto, &svc.Source, &svc.FirstSeen, &svc.LastSeen, &svc.Metadata)
		if err != nil {
			continue
		}
		services = append(services, svc)
	}

	return services, rows.Err()
}

func (s *Store) FindPort(port int, node string) (*PortUsage, error) {
	query := "SELECT node, port, name FROM services WHERE port = ?"
	var args []interface{}{port}

	if node != "" {
		query += " AND node = ?"
		args = append(args, node)
	}

	var result PortUsage
	var processName string
	err := s.db.QueryRow(query, args...).Scan(&result.Node, &result.Port, &processName)
	if err == sql.ErrNoRows {
		return &PortUsage{Port: port, InUse: false}, nil
	}
	if err != nil {
		return nil, err
	}

	result.InUse = true
	result.Process = processName
	return &result, nil
}

func (s *Store) SaveSnapshot(node string, data string) error {
	_, err := s.db.Exec("INSERT INTO snapshots (node, data) VALUES (?, ?)", node, data)
	return err
}

func (s *Store) Close() error {
	return s.db.Close()
}
