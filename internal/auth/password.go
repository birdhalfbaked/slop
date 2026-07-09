package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters. Kept in the PHC-encoded hash so they can evolve safely.
const (
	argonMemory      = 64 * 1024 // 64 MiB
	argonIterations  = 3
	argonParallelism = 2
	argonSaltLen     = 16
	argonKeyLen      = 32
)

// HashPassword returns an Argon2id PHC-formatted hash:
// $argon2id$v=19$m=65536,t=3,p=2$<salt>$<hash>
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonIterations, argonParallelism,
		b64.EncodeToString(salt), b64.EncodeToString(hash)), nil
}

// CheckPassword verifies a password against an Argon2id PHC-formatted hash.
func CheckPassword(encoded, password string) bool {
	memory, iterations, parallelism, salt, hash, err := decodeArgon2(encoded)
	if err != nil {
		return false
	}
	computed := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(hash)))
	return subtle.ConstantTimeCompare(hash, computed) == 1
}

func decodeArgon2(encoded string) (memory, iterations uint32, parallelism uint8, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return 0, 0, 0, nil, nil, errors.New("invalid argon2 hash")
	}
	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	if version != argon2.Version {
		return 0, 0, 0, nil, nil, errors.New("incompatible argon2 version")
	}
	var mem, iters, par int
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iters, &par); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	b64 := base64.RawStdEncoding
	if salt, err = b64.DecodeString(parts[4]); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	if hash, err = b64.DecodeString(parts[5]); err != nil {
		return 0, 0, 0, nil, nil, err
	}
	return uint32(mem), uint32(iters), uint8(par), salt, hash, nil
}
