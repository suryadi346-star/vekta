package cache

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB adalah wrapper cache SQLite dengan pool terkontrol.
type DB struct {
	conn *sql.DB
}

// NewSQLiteCache membuka koneksi SQLite dan mengkonfigurasi pool.
// FIX #1: SetMaxOpenConns(1) — SQLite WAL mode paling stabil dengan
// single writer. Multiple reader tetap bisa berjalan paralel via WAL.
// FIX #2: SetMaxIdleConns(1) + ConnMaxLifetime mencegah koneksi stale
// yang bisa menyebabkan lock phantom.
func NewSQLiteCache(path string) (*DB, error) {
	// DSN dengan pragma inline — lebih reliable daripada PRAGMA setelah open
	dsn := fmt.Sprintf(
		"%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache_size=-8000&_foreign_keys=on",
		path,
	)

	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// FIX: Pool control — SQLite tidak butuh banyak koneksi.
	// 1 writer, beberapa reader sudah cukup untuk beban ringan.
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(30 * time.Minute)
	conn.SetConnMaxIdleTime(5 * time.Minute)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// migrate membuat tabel jika belum ada.
func (db *DB) migrate() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS chat_cache (
			key       TEXT PRIMARY KEY,
			value     TEXT NOT NULL,
			expires_at INTEGER NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_cache_expires ON chat_cache(expires_at);
	`)
	return err
}

// Get mengambil nilai dari cache. Return ("", false) jika miss atau expired.
func (db *DB) Get(key string) (string, bool) {
	var value string
	var expiresAt int64

	err := db.conn.QueryRow(
		`SELECT value, expires_at FROM chat_cache WHERE key = ?`, key,
	).Scan(&value, &expiresAt)

	if err != nil {
		return "", false
	}

	if time.Now().Unix() > expiresAt {
		// Lazy delete — tidak block caller
		go db.conn.Exec(`DELETE FROM chat_cache WHERE key = ?`, key)
		return "", false
	}

	return value, true
}

// Set menyimpan nilai ke cache dengan TTL dalam menit.
func (db *DB) Set(key, value string, ttlMinutes int) error {
	expiresAt := time.Now().Add(time.Duration(ttlMinutes) * time.Minute).Unix()

	_, err := db.conn.Exec(
		`INSERT OR REPLACE INTO chat_cache (key, value, expires_at) VALUES (?, ?, ?)`,
		key, value, expiresAt,
	)
	return err
}

// Evict membersihkan entry expired — panggil periodik dari background goroutine.
func (db *DB) Evict() (int64, error) {
	res, err := db.conn.Exec(
		`DELETE FROM chat_cache WHERE expires_at < ?`, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Close menutup koneksi dengan bersih.
func (db *DB) Close() error {
	return db.conn.Close()
}
