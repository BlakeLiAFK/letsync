package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// GenerateRandomKey 生成随机密钥
func GenerateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateSignature 生成 HMAC-SHA256 签名
func GenerateSignature(uuid string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(uuid))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature 验证签名
func VerifySignature(uuid, signature, secret string) bool {
	expected := GenerateSignature(uuid, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// HashPassword 密码哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Encrypt AES 加密
func Encrypt(plaintext, key string) (string, error) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt AES 解密
func Decrypt(ciphertext, key string) (string, error) {
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// CertFingerprint 计算证书指纹
func CertFingerprint(certPEM []byte) (string, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(cert.Raw)
	return "sha256:" + hex.EncodeToString(hash[:]), nil
}
