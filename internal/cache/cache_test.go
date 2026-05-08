
package cache

import "fmt"

import (
    "os"
    "testing"
    "time"
)

func setupTestDB(t *testing.T) (*DB, func()) {
    t.Helper()
    tmp, err := os.CreateTemp("", "cache_test_*.db")
    if err != nil {
        t.Fatalf("create temp db: %v", err)
    }
    tmp.Close()

    db, err := NewSQLiteCache(tmp.Name())
    if err != nil {
        os.Remove(tmp.Name())
        t.Fatalf("new cache: %v", err)
    }
    return db, func() {
        db.Close()
        os.Remove(tmp.Name())
    }
}

func TestSetAndGet(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    if err := db.Set("k1", "hello world", 60); err != nil {
        t.Fatalf("set error: %v", err)
    }
    val, ok := db.Get("k1")
    if !ok {
        t.Fatal("expected cache hit, got miss")
    }
    if val != "hello world" {
        t.Errorf("wrong value: %q", val)
    }
}

func TestGetMiss(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    _, ok := db.Get("nonexistent")
    if ok {
        t.Fatal("expected cache miss, got hit")
    }
}

func TestExpiredEntry(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    // Set dengan TTL -1 menit (sudah expired sejak awal)
    _, err := db.conn.Exec(
        `INSERT OR REPLACE INTO chat_cache (key, value, expires_at) VALUES (?, ?, ?)`,
        "expired_key", "old_value", time.Now().Add(-1*time.Minute).Unix(),
    )
    if err != nil {
        t.Fatalf("manual insert: %v", err)
    }

    _, ok := db.Get("expired_key")
    if ok {
        t.Fatal("expected expired entry to be a miss")
    }
}

func TestOverwrite(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    db.Set("key", "v1", 60)
    db.Set("key", "v2", 60)

    val, ok := db.Get("key")
    if !ok {
        t.Fatal("expected hit")
    }
    if val != "v2" {
        t.Errorf("expected v2, got %q", val)
    }
}

func TestEvict(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    // insert 2 expired, 1 valid
    db.conn.Exec(`INSERT OR REPLACE INTO chat_cache VALUES ('e1','v1',?)`, time.Now().Add(-1*time.Minute).Unix())
    db.conn.Exec(`INSERT OR REPLACE INTO chat_cache VALUES ('e2','v2',?)`, time.Now().Add(-2*time.Minute).Unix())
    db.Set("valid", "keep", 60)

    n, err := db.Evict()
    if err != nil {
        t.Fatalf("evict error: %v", err)
    }
    if n != 2 {
        t.Errorf("expected 2 evicted, got %d", n)
    }

    _, ok := db.Get("valid")
    if !ok {
        t.Fatal("valid entry should survive eviction")
    }
}

func TestConcurrentSet(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()

    done := make(chan struct{}, 20)
    for i := 0; i < 20; i++ {
        go func(n int) {
            defer func() { done <- struct{}{} }()
            // BUG TARGET: SQLite dengan MaxOpenConns(1) — tidak boleh SQLITE_BUSY
            key := fmt.Sprintf("key_%d", n)
            db.Set(key, "value", 60)
        }(i)
    }
    for i := 0; i < 20; i++ {
        <-done
    }
    // Kalau sampai sini tanpa panic/deadlock = PASS
}
