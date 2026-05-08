
package cache

import (
    "fmt"
    "os"
    "testing"
)

func BenchmarkSet(b *testing.B) {
    tmp, _ := os.CreateTemp("", "bench_*.db")
    tmp.Close()
    defer os.Remove(tmp.Name())
    db, _ := NewSQLiteCache(tmp.Name())
    defer db.Close()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        db.Set(fmt.Sprintf("k%d", i), "value", 60)
    }
}

func BenchmarkGet(b *testing.B) {
    tmp, _ := os.CreateTemp("", "bench_*.db")
    tmp.Close()
    defer os.Remove(tmp.Name())
    db, _ := NewSQLiteCache(tmp.Name())
    defer db.Close()
    db.Set("fixed_key", "fixed_value", 60)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        db.Get("fixed_key")
    }
}
