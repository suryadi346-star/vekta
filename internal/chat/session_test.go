package chat

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSessionAdd(t *testing.T) {
	s := newSession("test-1", 4)
	s.Add("user", "hello")
	s.Add("assistant", "hi")

	if len(s.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(s.Messages))
	}
}

func TestSessionTrim_NoSystemPrompt(t *testing.T) {
	s := newSession("trim-1", 3)
	s.Add("user", "msg1")
	s.Add("assistant", "reply1")
	s.Add("user", "msg2")
	// ini harusnya trigger trim — buang msg1
	s.Add("assistant", "reply2")

	if len(s.Messages) != 3 {
		t.Errorf("expected 3 after trim, got %d", len(s.Messages))
	}
	if s.Messages[0].Content != "reply1" {
		t.Errorf("expected oldest to be trimmed, got %q", s.Messages[0].Content)
	}
}

func TestSessionTrim_WithSystemPrompt(t *testing.T) {
	s := newSession("trim-2", 3)
	s.Add("system", "you are vekta")
	s.Add("user", "msg1")
	s.Add("assistant", "reply1")
	// trigger trim — system harus tetap ada
	s.Add("user", "msg2")

	if s.Messages[0].Role != "system" {
		t.Errorf("system prompt harus tetap di index 0, got role=%q", s.Messages[0].Role)
	}
	if len(s.Messages) != 3 {
		t.Errorf("expected 3 after trim, got %d", len(s.Messages))
	}
}

func TestSessionHistory_IsCopy(t *testing.T) {
	s := newSession("copy-1", 10)
	s.Add("user", "hello")

	h := s.History()
	h[0].Content = "tampered"

	if s.Messages[0].Content == "tampered" {
		t.Error("History() harus return copy, bukan reference")
	}
}

func TestManagerGetOrCreate(t *testing.T) {
	m := NewManager(5*time.Minute, 10)

	s1 := m.Get("session-a")
	s2 := m.Get("session-a")

	if s1 != s2 {
		t.Error("Get dengan ID sama harus return instance yang sama")
	}
}

func TestManagerDelete(t *testing.T) {
	m := NewManager(5*time.Minute, 10)
	m.Get("del-me")
	m.Delete("del-me")

	if m.Count() != 0 {
		t.Errorf("expected 0 sessions after delete, got %d", m.Count())
	}
}

func TestManagerConcurrent(t *testing.T) {
	m := NewManager(5*time.Minute, 20)
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("session-%d", n%10)
			s := m.Get(id)
			s.Add("user", fmt.Sprintf("msg from goroutine %d", n))
		}(i)
	}
	wg.Wait()
	// kalau sampai sini tanpa panic/deadlock = PASS
}
