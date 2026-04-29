// Package crypto 提供 Ed25519 密钥管理、签名与验证功能。
package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// GenerateKeyPair 生成 Ed25519 密钥对并保存为 PEM 文件。
func GenerateKeyPair(_ int, privPath, pubPath string) error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	})

	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("marshal public key: %w", err)
	}

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	if err := os.WriteFile(pubPath, pubPEM, 0644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}

	return nil
}

// LoadPrivateKey 从 PEM 文件加载 Ed25519 私钥。
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	edKey, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not Ed25519 private key")
	}

	return edKey, nil
}

// LoadPublicKey 从 PEM 文件加载 Ed25519 公钥。
func LoadPublicKey(path string) (ed25519.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	edPub, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not Ed25519 public key")
	}

	return edPub, nil
}

// LoadPublicKeyFromBytes 从 PEM 字节加载 Ed25519 公钥。
func LoadPublicKeyFromBytes(pemData []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	edPub, ok := pub.(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not Ed25519 public key")
	}

	return edPub, nil
}

// Sign 使用 Ed25519 私钥对数据进行签名。
func Sign(privateKey ed25519.PrivateKey, data []byte) ([]byte, error) {
	signature := ed25519.Sign(privateKey, data)
	return signature, nil
}

// Verify 使用 Ed25519 公钥验证签名。
func Verify(publicKey ed25519.PublicKey, data, signature []byte) error {
	if !ed25519.Verify(publicKey, data, signature) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

// EnsureKeyPair 确保密钥对存在，不存在则自动生成。
func EnsureKeyPair(bits int, privPath, pubPath string) (ed25519.PrivateKey, error) {
	if _, err := os.Stat(privPath); os.IsNotExist(err) {
		if err := os.MkdirAll("keys", 0700); err != nil {
			return nil, err
		}
		if err := GenerateKeyPair(bits, privPath, pubPath); err != nil {
			return nil, err
		}
	}
	return LoadPrivateKey(privPath)
}
