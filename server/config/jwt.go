package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

var jwtSecretMu sync.RWMutex

// RegenerateSecretKey regenerate secret key when logout
func RegenerateSecretKey() {
	jwtSecretMu.Lock()
	defer jwtSecretMu.Unlock()
	if instance.JWT.RevokeTokensOnLogout {
		instance.JWT.SecretKey = generateRandomSecretKey()
	}
}

func GetJWTSecretKey() string {
	jwtSecretMu.RLock()
	defer jwtSecretMu.RUnlock()
	return instance.JWT.SecretKey
}

// Generate random string for secret key.
func generateRandomSecretKey() string {
	b := make([]byte, 64)
	_, err := rand.Read(b)
	if err != nil {
		currentTime := time.Now().UnixNano()
		timeString := fmt.Sprintf("%d", currentTime)
		return fmt.Sprintf("%064s", timeString)
	}

	return base64.URLEncoding.EncodeToString(b)
}
