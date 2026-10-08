package memory

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestRememberSearchScopeAndForget(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "memory.aflite")
	alice := Context{AgentID: "support", UserID: "alice", Workspace: "/project", SessionID: "s1"}
	bob := Context{AgentID: "support", UserID: "bob", Workspace: "/other", SessionID: "s2"}
	record, err := remember(dbPath, Request{
		Text: "Alice prefers concise weekly status reports", Kind: "preference", Scope: "user",
		Importance: 0.8, Context: alice, Source: "explicit",
	})
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := remember(dbPath, Request{
		Text: "Alice prefers concise weekly status reports", Kind: "preference", Scope: "user",
		Importance: 0.8, Context: Context{AgentID: "support", UserID: "alice", Workspace: "/other", SessionID: "s2"}, Source: "explicit",
	})
	if err != nil || duplicate.ID != record.ID || duplicate.CreatedAt != record.CreatedAt {
		t.Fatalf("idempotent write failed: record=%+v duplicate=%+v err=%v", record, duplicate, err)
	}
	aliceResult, err := search(dbPath, Request{Query: "weekly status reports", Limit: 6, Context: alice})
	if err != nil || len(aliceResult.Hits) != 1 || aliceResult.Hits[0].ID != record.ID {
		t.Fatalf("Alice search failed: %+v err=%v", aliceResult, err)
	}
	bobResult, err := search(dbPath, Request{Query: "weekly status reports", Limit: 6, Context: bob})
	if err != nil || len(bobResult.Hits) != 0 {
		t.Fatalf("user-scoped memory leaked: %+v err=%v", bobResult, err)
	}
	if _, err := forget(dbPath, Request{ID: record.ID, Context: bob}); err == nil {
		t.Fatal("expected cross-user delete to fail")
	}
	deleted, err := forget(dbPath, Request{ID: record.ID, Context: alice})
	if err != nil || deleted["deleted"] != true {
		t.Fatalf("delete failed: %+v err=%v", deleted, err)
	}
}

func TestScopeRequiresIdentity(t *testing.T) {
	_, err := remember(filepath.Join(t.TempDir(), "memory.aflite"), Request{
		Text: "secret", Scope: "user", Context: Context{},
	})
	if err == nil {
		t.Fatal("expected user scope without user identity to fail")
	}
}

func TestForgetTextUsesExactScopedIdentity(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "memory.aflite")
	alice := Context{AgentID: "support", UserID: "alice", SessionID: "s1"}
	bob := Context{AgentID: "support", UserID: "bob", SessionID: "s2"}
	request := Request{
		Text: "Alice prefers concise replies", Kind: "user_profile", Scope: "user", Context: alice,
	}
	record, err := remember(dbPath, request)
	if err != nil {
		t.Fatal(err)
	}
	wrongUser, err := forgetText(dbPath, Request{
		Text: request.Text, Kind: request.Kind, Scope: request.Scope, Context: bob,
	})
	if err != nil || wrongUser["deleted"] != false {
		t.Fatalf("cross-user exact deletion was not isolated: %+v err=%v", wrongUser, err)
	}
	deleted, err := forgetText(dbPath, request)
	if err != nil || deleted["deleted"] != true || deleted["id"] != record.ID {
		t.Fatalf("exact deletion failed: %+v err=%v", deleted, err)
	}
}

func TestConcurrentFirstUse(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "memory.aflite")
	start := make(chan struct{})
	errorsByWriter := make([]error, 4)
	var writers sync.WaitGroup
	for index := range errorsByWriter {
		writers.Add(1)
		go func() {
			defer writers.Done()
			<-start
			_, errorsByWriter[index] = remember(dbPath, Request{
				Text: "concurrent memory", Scope: "profile", Context: Context{},
			})
		}()
	}
	close(start)
	writers.Wait()
	for index, err := range errorsByWriter {
		if err != nil {
			t.Fatalf("writer %d failed: %v", index, err)
		}
	}
}
