package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"sync"
	"time"
)

const (
	oidcTransactionLifetime       = 5 * time.Minute
	maxOIDCTransactions           = 128
	maxOIDCTransactionsPerBinding = 4
)

type oidcTransaction struct {
	nonce        string
	pkceVerifier string
	bindingHash  [sha256.Size]byte
	createdAt    time.Time
	expiresAt    time.Time
}

type oidcStateStore struct {
	mu           sync.Mutex
	transactions map[string]oidcTransaction
	now          func() time.Time
}

func newOIDCStateStore() *oidcStateStore {
	return &oidcStateStore{transactions: make(map[string]oidcTransaction), now: time.Now}
}

func (s *oidcStateStore) create(binding string) (state, nonce, verifier, challenge string, err error) {
	if binding == "" {
		return "", "", "", "", errors.New("OIDC browser binding is required")
	}
	state, err = randomURLValue(32)
	if err != nil {
		return "", "", "", "", err
	}
	nonce, err = randomURLValue(32)
	if err != nil {
		return "", "", "", "", err
	}
	verifier, err = randomURLValue(64)
	if err != nil {
		return "", "", "", "", err
	}
	digest := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(digest[:])
	bindingHash := sha256.Sum256([]byte(binding))

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
	if s.countBindingLocked(bindingHash) >= maxOIDCTransactionsPerBinding {
		s.evictOldestBindingLocked(bindingHash)
	} else if len(s.transactions) >= maxOIDCTransactions {
		return "", "", "", "", errors.New("too many pending OIDC logins")
	}
	s.transactions[state] = oidcTransaction{
		nonce: nonce, pkceVerifier: verifier, bindingHash: bindingHash,
		createdAt: s.now(), expiresAt: s.now().Add(oidcTransactionLifetime),
	}
	return state, nonce, verifier, challenge, nil
}

func (s *oidcStateStore) consume(state, binding string) (oidcTransaction, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
	txn, ok := s.transactions[state]
	bindingHash := sha256.Sum256([]byte(binding))
	if ok && subtle.ConstantTimeCompare(txn.bindingHash[:], bindingHash[:]) == 1 {
		delete(s.transactions, state)
		return txn, true
	}
	return oidcTransaction{}, false
}

func (s *oidcStateStore) countBindingLocked(bindingHash [sha256.Size]byte) int {
	count := 0
	for _, txn := range s.transactions {
		if subtle.ConstantTimeCompare(txn.bindingHash[:], bindingHash[:]) == 1 {
			count++
		}
	}
	return count
}

func (s *oidcStateStore) evictOldestBindingLocked(bindingHash [sha256.Size]byte) {
	var oldestState string
	var oldestTime time.Time
	for state, txn := range s.transactions {
		if subtle.ConstantTimeCompare(txn.bindingHash[:], bindingHash[:]) == 1 &&
			(oldestState == "" || txn.createdAt.Before(oldestTime)) {
			oldestState = state
			oldestTime = txn.createdAt
		}
	}
	delete(s.transactions, oldestState)
}

func (s *oidcStateStore) pruneLocked() {
	now := s.now()
	for state, txn := range s.transactions {
		if !txn.expiresAt.After(now) {
			delete(s.transactions, state)
		}
	}
}

func randomURLValue(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}
