package tests

import (
	"os"
	"testing"
	"time"

	sw "github.com/d-callan/service-datamonkey/go"
)

func TestSessionStore(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test_sessions_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	// Create session store
	store, err := sw.NewSQLiteSessionTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}
	defer store.Close()

	t.Run("Create session", func(t *testing.T) {
		session, err := store.CreateSession()
		if err != nil {
			t.Errorf("Failed to create session: %v", err)
		}

		if session.Subject == "" {
			t.Error("Session subject is empty")
		}

		if session.CreatedAt.IsZero() {
			t.Error("CreatedAt not set")
		}

		if session.LastSeen.IsZero() {
			t.Error("LastSeen not set")
		}
	})

	t.Run("Get session", func(t *testing.T) {
		// Create session
		created, err := store.CreateSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Get session
		retrieved, err := store.GetSession(created.Subject)
		if err != nil {
			t.Errorf("Failed to get session: %v", err)
		}

		if retrieved.Subject != created.Subject {
			t.Errorf("Subject mismatch: got %s, want %s", retrieved.Subject, created.Subject)
		}
	})

	t.Run("Get non-existent session", func(t *testing.T) {
		_, err := store.GetSession("non-existent-subject")
		if err == nil {
			t.Error("Expected error for non-existent session")
		}
	})

	t.Run("Update last seen", func(t *testing.T) {
		// Create session
		session, err := store.CreateSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		originalLastSeen := session.LastSeen

		// Wait at least 1 second so Unix timestamp changes (SQLite stores as INTEGER seconds)
		time.Sleep(1100 * time.Millisecond)

		// Update last seen
		err = store.UpdateLastSeen(session.Subject)
		if err != nil {
			t.Errorf("Failed to update last seen: %v", err)
		}

		// Get session again
		updated, err := store.GetSession(session.Subject)
		if err != nil {
			t.Errorf("Failed to get session: %v", err)
		}

		// Check that LastSeen was updated (Unix seconds should be at least 1 second later)
		if updated.LastSeen.Unix() <= originalLastSeen.Unix() {
			t.Errorf("LastSeen was not updated: original=%v, updated=%v",
				originalLastSeen.Unix(), updated.LastSeen.Unix())
		}
	})

	t.Run("Update last seen for non-existent session", func(t *testing.T) {
		err := store.UpdateLastSeen("non-existent-subject")
		if err == nil {
			t.Error("Expected error for non-existent session")
		}
	})

	t.Run("Delete session", func(t *testing.T) {
		// Create session
		session, err := store.CreateSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Delete session
		err = store.DeleteSession(session.Subject)
		if err != nil {
			t.Errorf("Failed to delete session: %v", err)
		}

		// Try to get deleted session
		_, err = store.GetSession(session.Subject)
		if err == nil {
			t.Error("Expected error for deleted session")
		}
	})

	t.Run("Delete non-existent session", func(t *testing.T) {
		err := store.DeleteSession("non-existent-subject")
		if err == nil {
			t.Error("Expected error for non-existent session")
		}
	})

	t.Run("Cleanup expired sessions", func(t *testing.T) {
		// Create some sessions
		session1, _ := store.CreateSession()
		session2, _ := store.CreateSession()
		session3, _ := store.CreateSession()

		// Update last seen for session1 and session2 to be old
		// We need to directly update the database for this test
		// In a real scenario, sessions would naturally age

		// For now, just test that cleanup runs without error
		count, err := store.CleanupExpiredSessions(24 * time.Hour)
		if err != nil {
			t.Errorf("Failed to cleanup expired sessions: %v", err)
		}

		// Should not have cleaned up any sessions (they're all new)
		if count > 0 {
			t.Errorf("Unexpected cleanup count: %d", count)
		}

		// Verify sessions still exist
		_, err = store.GetSession(session1.Subject)
		if err != nil {
			t.Error("Session1 should still exist")
		}

		_, err = store.GetSession(session2.Subject)
		if err != nil {
			t.Error("Session2 should still exist")
		}

		_, err = store.GetSession(session3.Subject)
		if err != nil {
			t.Error("Session3 should still exist")
		}
	})

	t.Run("Cleanup with very short max age", func(t *testing.T) {
		// Create session
		session, _ := store.CreateSession()

		// Wait at least 2 seconds to ensure Unix timestamp is definitely old enough
		// (SQLite stores timestamps as INTEGER seconds)
		time.Sleep(2100 * time.Millisecond)

		// Cleanup sessions older than 1 second (session is 2+ seconds old)
		count, err := store.CleanupExpiredSessions(1 * time.Second)
		if err != nil {
			t.Errorf("Failed to cleanup: %v", err)
		}

		if count == 0 {
			t.Error("Expected to cleanup at least one session")
		}

		// Session should be gone
		_, err = store.GetSession(session.Subject)
		if err == nil {
			t.Error("Session should have been cleaned up")
		}
	})

	t.Run("Multiple sessions", func(t *testing.T) {
		// Create multiple sessions
		sessions := make([]*sw.Session, 5)
		for i := 0; i < 5; i++ {
			session, err := store.CreateSession()
			if err != nil {
				t.Fatalf("Failed to create session %d: %v", i, err)
			}
			sessions[i] = session
		}

		// Verify all sessions exist and have unique subjects
		subjectMap := make(map[string]bool)
		for i, session := range sessions {
			retrieved, err := store.GetSession(session.Subject)
			if err != nil {
				t.Errorf("Failed to get session %d: %v", i, err)
			}

			if subjectMap[retrieved.Subject] {
				t.Errorf("Duplicate subject found: %s", retrieved.Subject)
			}
			subjectMap[retrieved.Subject] = true
		}
	})
}

func TestSessionStoreConcurrency(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test_sessions_concurrent_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	// Create session store
	store, err := sw.NewSQLiteSessionTracker(dbPath)
	if err != nil {
		t.Fatalf("Failed to create session store: %v", err)
	}
	defer store.Close()

	t.Run("Concurrent session creation", func(t *testing.T) {
		done := make(chan bool)
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := store.CreateSession()
				if err != nil {
					t.Errorf("Failed to create session: %v", err)
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	t.Run("Concurrent updates", func(t *testing.T) {
		// Create session
		session, err := store.CreateSession()
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		done := make(chan bool)
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			go func() {
				err := store.UpdateLastSeen(session.Subject)
				if err != nil {
					t.Errorf("Failed to update last seen: %v", err)
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify session still exists
		_, err = store.GetSession(session.Subject)
		if err != nil {
			t.Error("Session should still exist after concurrent updates")
		}
	})
}
