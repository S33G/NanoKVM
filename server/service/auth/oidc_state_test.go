package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"
)

func TestOIDCStateStorePKCEAndSingleUse(t *testing.T) {
	store := newOIDCStateStore()
	state, nonce, verifier, challenge, err := store.create("browser-binding")
	if err != nil {
		t.Fatal(err)
	}
	if state == "" || nonce == "" || verifier == "" || state == nonce {
		t.Fatal("create() returned missing or reused random transaction values")
	}
	digest := sha256.Sum256([]byte(verifier))
	if want := base64.RawURLEncoding.EncodeToString(digest[:]); challenge != want {
		t.Fatal("PKCE challenge does not use S256")
	}
	if _, ok := store.consume(state, "different-browser"); ok {
		t.Fatal("state was accepted from a different browser binding")
	}
	txn, ok := store.consume(state, "browser-binding")
	if !ok || txn.nonce != nonce || txn.pkceVerifier != verifier {
		t.Fatal("consume() did not return the created transaction")
	}
	if _, ok := store.consume(state, "browser-binding"); ok {
		t.Fatal("state was accepted more than once")
	}
}

func TestOIDCStateStoreExpiry(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newOIDCStateStore()
	store.now = func() time.Time { return now }
	state, _, _, _, err := store.create("browser-binding")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(oidcTransactionLifetime)
	if _, ok := store.consume(state, "browser-binding"); ok {
		t.Fatal("state was accepted at its expiry time")
	}
}

func TestOIDCStateStoreCapacityAndPruning(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	store := newOIDCStateStore()
	store.now = func() time.Time { return now }
	var bindingOldest string
	for i := 0; i < maxOIDCTransactionsPerBinding; i++ {
		state, _, _, _, err := store.create("one-browser")
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			bindingOldest = state
		}
		now = now.Add(time.Millisecond)
	}
	if _, _, _, _, err := store.create("one-browser"); err != nil {
		t.Fatalf("per-binding eviction failed: %v", err)
	}
	if _, ok := store.consume(bindingOldest, "one-browser"); ok {
		t.Fatal("oldest same-browser transaction was not evicted")
	}

	store = newOIDCStateStore()
	store.now = func() time.Time { return now }
	var oldestState string
	for i := 0; i < maxOIDCTransactions; i++ {
		binding := fmt.Sprintf("browser-%d", i)
		state, _, _, _, err := store.create(binding)
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		if i == 0 {
			oldestState = state
		}
		now = now.Add(time.Millisecond)
	}
	if _, _, _, _, err := store.create("new-browser"); err == nil {
		t.Fatal("global transaction capacity was exceeded")
	}
	if _, ok := store.consume(oldestState, "browser-0"); !ok {
		t.Fatal("a different browser evicted an existing transaction")
	}
	now = now.Add(oidcTransactionLifetime)
	if _, _, _, _, err := store.create("browser-binding"); err != nil {
		t.Fatalf("create() did not prune expired transactions: %v", err)
	}
}

func TestOIDCStateStoreRequiresBrowserBinding(t *testing.T) {
	if _, _, _, _, err := newOIDCStateStore().create(""); err == nil {
		t.Fatal("create() accepted an empty browser binding")
	}
}

func TestOIDCLoginRateLimit(t *testing.T) {
	s := newOIDCService()
	now := time.Unix(1_700_000_000, 0)
	for i := 0; i < maxOIDCLoginsPerWindow; i++ {
		if !s.allowLogin("192.0.2.1", now) {
			t.Fatalf("login %d was rate limited too early", i)
		}
	}
	if s.allowLogin("192.0.2.1", now) {
		t.Fatal("peer exceeded OIDC login rate limit")
	}
	if !s.allowLogin("192.0.2.2", now) {
		t.Fatal("one peer rate limited a different peer")
	}
	if !s.allowLogin("192.0.2.1", now.Add(oidcLoginRateWindow)) {
		t.Fatal("rate limit did not reset after its window")
	}
}
