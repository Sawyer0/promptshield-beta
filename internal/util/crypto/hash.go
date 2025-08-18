package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// HashAlgorithm represents supported hash algorithms
type HashAlgorithm string

const (
	AlgorithmSHA256 HashAlgorithm = "sha256"
	AlgorithmSHA512 HashAlgorithm = "sha512"
	AlgorithmSHA1   HashAlgorithm = "sha1"
	AlgorithmMD5    HashAlgorithm = "md5"
)

// Hash computes hash using specified algorithm
func Hash(data []byte, algorithm HashAlgorithm) ([]byte, error) {
	var h hash.Hash
	
	switch algorithm {
	case AlgorithmSHA256:
		h = sha256.New()
	case AlgorithmSHA512:
		h = sha512.New()
	case AlgorithmSHA1:
		h = sha1.New()
	case AlgorithmMD5:
		h = md5.New()
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
	
	h.Write(data)
	return h.Sum(nil), nil
}

// HashString computes hash of string and returns hex encoding
func HashString(data string, algorithm HashAlgorithm) (string, error) {
	hash, err := Hash([]byte(data), algorithm)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// SHA256 computes SHA256 hash
func SHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// SHA256String computes SHA256 hash and returns hex string
func SHA256String(data string) string {
	hash := SHA256([]byte(data))
	return hex.EncodeToString(hash)
}

// SHA512 computes SHA512 hash
func SHA512(data []byte) []byte {
	hash := sha512.Sum512(data)
	return hash[:]
}

// SHA512String computes SHA512 hash and returns hex string
func SHA512String(data string) string {
	hash := SHA512([]byte(data))
	return hex.EncodeToString(hash)
}

// MD5 computes MD5 hash (not recommended for security)
func MD5(data []byte) []byte {
	hash := md5.Sum(data)
	return hash[:]
}

// MD5String computes MD5 hash and returns hex string
func MD5String(data string) string {
	hash := MD5([]byte(data))
	return hex.EncodeToString(hash)
}

// GenerateSalt generates a random salt
func GenerateSalt(length int) ([]byte, error) {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	return salt, err
}

// GenerateSaltString generates a random salt as base64 string
func GenerateSaltString(length int) (string, error) {
	salt, err := GenerateSalt(length)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

// PBKDF2 derives key using PBKDF2
func PBKDF2(password, salt []byte, iterations, keyLength int, hashFunc func() hash.Hash) []byte {
	return pbkdf2.Key(password, salt, iterations, keyLength, hashFunc)
}

// PBKDF2SHA256 derives key using PBKDF2 with SHA256
func PBKDF2SHA256(password, salt []byte, iterations, keyLength int) []byte {
	return PBKDF2(password, salt, iterations, keyLength, sha256.New)
}

// PBKDF2SHA512 derives key using PBKDF2 with SHA512
func PBKDF2SHA512(password, salt []byte, iterations, keyLength int) []byte {
	return PBKDF2(password, salt, iterations, keyLength, sha512.New)
}

// ScryptParams represents scrypt parameters
type ScryptParams struct {
	N      int // CPU/memory cost parameter
	R      int // Block size parameter
	P      int // Parallelization parameter
	KeyLen int // Key length
}

// DefaultScryptParams returns recommended scrypt parameters
func DefaultScryptParams() *ScryptParams {
	return &ScryptParams{
		N:      32768, // 2^15
		R:      8,
		P:      1,
		KeyLen: 32,
	}
}

// Scrypt derives key using scrypt
func Scrypt(password, salt []byte, params *ScryptParams) ([]byte, error) {
	if params == nil {
		params = DefaultScryptParams()
	}
	return scrypt.Key(password, salt, params.N, params.R, params.P, params.KeyLen)
}

// Argon2Params represents Argon2 parameters
type Argon2Params struct {
	Time    uint32 // Number of iterations
	Memory  uint32 // Memory usage in KiB
	Threads uint8  // Number of threads
	KeyLen  uint32 // Key length
}

// DefaultArgon2Params returns recommended Argon2 parameters
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Time:    1,
		Memory:  64 * 1024, // 64 MB
		Threads: 4,
		KeyLen:  32,
	}
}

// Argon2ID derives key using Argon2id
func Argon2ID(password, salt []byte, params *Argon2Params) []byte {
	if params == nil {
		params = DefaultArgon2Params()
	}
	return argon2.IDKey(password, salt, params.Time, params.Memory, params.Threads, params.KeyLen)
}

// Argon2I derives key using Argon2i
func Argon2I(password, salt []byte, params *Argon2Params) []byte {
	if params == nil {
		params = DefaultArgon2Params()
	}
	return argon2.Key(password, salt, params.Time, params.Memory, params.Threads, params.KeyLen)
}

// BcryptCost represents bcrypt cost parameter
type BcryptCost int

const (
	BcryptMinCost     BcryptCost = 4
	BcryptMaxCost     BcryptCost = 31
	BcryptDefaultCost BcryptCost = 10
)

// BcryptHash hashes password using bcrypt
func BcryptHash(password string, cost BcryptCost) (string, error) {
	if cost < BcryptMinCost || cost > BcryptMaxCost {
		cost = BcryptDefaultCost
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), int(cost))
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// BcryptVerify verifies password against bcrypt hash
func BcryptVerify(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// HMAC computes HMAC using specified hash function
func HMAC(key, data []byte, hashFunc func() hash.Hash) []byte {
	mac := hmac.New(hashFunc, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// HMACSHA256 computes HMAC-SHA256
func HMACSHA256(key, data []byte) []byte {
	return HMAC(key, data, sha256.New)
}

// HMACSHA512 computes HMAC-SHA512
func HMACSHA512(key, data []byte) []byte {
	return HMAC(key, data, sha512.New)
}

// HMACSHA256String computes HMAC-SHA256 and returns hex string
func HMACSHA256String(key, data []byte) string {
	mac := HMACSHA256(key, data)
	return hex.EncodeToString(mac)
}

// HMACSHA512String computes HMAC-SHA512 and returns hex string
func HMACSHA512String(key, data []byte) string {
	mac := HMACSHA512(key, data)
	return hex.EncodeToString(mac)
}

// VerifyHMAC verifies HMAC in constant time
func VerifyHMAC(expected, actual []byte) bool {
	return hmac.Equal(expected, actual)
}

// HashFile computes hash of file
func HashFile(filepath string, algorithm HashAlgorithm) ([]byte, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var h hash.Hash
	switch algorithm {
	case AlgorithmSHA256:
		h = sha256.New()
	case AlgorithmSHA512:
		h = sha512.New()
	case AlgorithmSHA1:
		h = sha1.New()
	case AlgorithmMD5:
		h = md5.New()
	default:
		return nil, fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
	
	if _, err := io.Copy(h, file); err != nil {
		return nil, err
	}
	
	return h.Sum(nil), nil
}

// HashFileString computes hash of file and returns hex string
func HashFileString(filepath string, algorithm HashAlgorithm) (string, error) {
	hash, err := HashFile(filepath, algorithm)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash), nil
}

// CompareHash compares two hashes in constant time
func CompareHash(hash1, hash2 []byte) bool {
	return hmac.Equal(hash1, hash2)
}

// CompareHashString compares two hex-encoded hashes in constant time
func CompareHashString(hash1, hash2 string) bool {
	h1, err1 := hex.DecodeString(hash1)
	h2, err2 := hex.DecodeString(hash2)
	if err1 != nil || err2 != nil {
		return false
	}
	return CompareHash(h1, h2)
}

// RandomBytes generates cryptographically secure random bytes
func RandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	return bytes, err
}

// RandomHex generates random hex string
func RandomHex(length int) (string, error) {
	bytes, err := RandomBytes(length)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// RandomBase64 generates random base64 string
func RandomBase64(length int) (string, error) {
	bytes, err := RandomBytes(length)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}