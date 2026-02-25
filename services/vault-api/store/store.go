package store

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Evidence struct {
	ID        string
	LeafIndex *int64
}

type AuditEntry struct {
	ID         string
	ResourceID string
	Actor      string
	Timestamp  time.Time
}

type Store interface {
	SaveEvidence(ctx context.Context, id string) error
	AssignNextPendingLeaf(ctx context.Context) (*Evidence, error)
	GetEvidence(ctx context.Context, id string) (*Evidence, error)
	SaveAudit(ctx context.Context, e AuditEntry) error
	ListAudits(ctx context.Context, limit int) ([]AuditEntry, error)
}

var (
	mu      sync.Mutex
	current Store
)

func Current() Store {
	mu.Lock()
	defer mu.Unlock()
	return current
}

func Init(ctx context.Context) (Store, error) {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		return current, nil
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// fallback to in-memory
		mem := NewMemoryStore()
		current = mem
		return current, nil
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	pg := &pgStore{pool: pool}
	if err := pg.ensureSchema(ctx); err != nil {
		return nil, err
	}
	current = pg
	return current, nil
}

// -- memory store (fallback)
type memStore struct {
	mu     sync.Mutex
	ev     map[string]*Evidence
	audits []AuditEntry
	next   int64
}

func NewMemoryStore() *memStore {
	return &memStore{ev: map[string]*Evidence{}, audits: []AuditEntry{}, next: 0}
}

func (m *memStore) SaveEvidence(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ev[id] = &Evidence{ID: id, LeafIndex: nil}
	return nil
}

func (m *memStore) AssignNextPendingLeaf(ctx context.Context) (*Evidence, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.ev {
		if e.LeafIndex == nil {
			idx := m.next
			m.next++
			e.LeafIndex = &idx
			return e, nil
		}
	}
	return nil, nil
}

func (m *memStore) GetEvidence(ctx context.Context, id string) (*Evidence, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.ev[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return e, nil
}

func (m *memStore) SaveAudit(ctx context.Context, a AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audits = append(m.audits, a)
	return nil
}

func (m *memStore) ListAudits(ctx context.Context, limit int) ([]AuditEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 || limit > len(m.audits) {
		limit = len(m.audits)
	}
	res := make([]AuditEntry, 0, limit)
	for i := 0; i < limit; i++ {
		res = append(res, m.audits[i])
	}
	return res, nil
}

// -- pg store
type pgStore struct {
	pool *pgxpool.Pool
}

func (p *pgStore) ensureSchema(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
    CREATE TABLE IF NOT EXISTS evidence (
        id UUID PRIMARY KEY,
        leaf_index bigint NULL,
        created_at timestamptz DEFAULT now()
    );
    CREATE TABLE IF NOT EXISTS audit (
        id UUID PRIMARY KEY,
        resource_id UUID,
        actor TEXT,
        timestamp timestamptz
    );
    `)
	return err
}

func (p *pgStore) SaveEvidence(ctx context.Context, id string) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO evidence (id) VALUES ($1) ON CONFLICT DO NOTHING`, id)
	return err
}

func (p *pgStore) AssignNextPendingLeaf(ctx context.Context) (*Evidence, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	// pick one pending record
	var id string
	err = tx.QueryRow(ctx, `SELECT id FROM evidence WHERE leaf_index IS NULL ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&id)
	if err != nil {
		return nil, nil
	}
	// compute max leaf
	var maxLeaf *int64
	var tmp int64
	err = tx.QueryRow(ctx, `SELECT max(leaf_index) FROM evidence`).Scan(&maxLeaf)
	if err != nil {
		return nil, err
	}
	if maxLeaf == nil {
		tmp = 0
	} else {
		tmp = *maxLeaf + 1
	}
	_, err = tx.Exec(ctx, `UPDATE evidence SET leaf_index=$1 WHERE id=$2`, tmp, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	li := tmp
	return &Evidence{ID: id, LeafIndex: &li}, nil
}

func (p *pgStore) GetEvidence(ctx context.Context, id string) (*Evidence, error) {
	var li *int64
	err := p.pool.QueryRow(ctx, `SELECT leaf_index FROM evidence WHERE id=$1`, id).Scan(&li)
	if err != nil {
		return nil, err
	}
	return &Evidence{ID: id, LeafIndex: li}, nil
}

func (p *pgStore) SaveAudit(ctx context.Context, a AuditEntry) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	_, err := p.pool.Exec(ctx, `INSERT INTO audit (id, resource_id, actor, timestamp) VALUES ($1,$2,$3,$4)`, a.ID, a.ResourceID, a.Actor, a.Timestamp)
	return err
}

func (p *pgStore) ListAudits(ctx context.Context, limit int) ([]AuditEntry, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, resource_id, actor, timestamp FROM audit ORDER BY timestamp DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []AuditEntry
	for rows.Next() {
		var a AuditEntry
		if err := rows.Scan(&a.ID, &a.ResourceID, &a.Actor, &a.Timestamp); err != nil {
			return nil, err
		}
		res = append(res, a)
	}
	return res, nil
}
