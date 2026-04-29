// Package token 提供 License Token 的签名、编码和解码功能。
package token

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"cerberus.dev/pkg/crypto"
)

// Payload Token 载荷结构。
type Payload struct {
	ID               string `json:"id"`
	Product          string `json:"product"`
	Issuer           string `json:"issuer"`
	Mode             string `json:"mode"`
	MaxMachines      int    `json:"max_machines"`
	ValidFrom        int64  `json:"valid_from"`
	ValidUntil       int64  `json:"valid_until"`
	DurationSec      int64  `json:"duration_sec"`
	IPBindingEnabled bool   `json:"ip_binding_enabled"`
	Fingerprint      string `json:"fingerprint,omitempty"`
	TokenGen         int    `json:"token_gen"`
	EnvHash          string `json:"env_hash,omitempty"`
}

// LicenseToken Token 结构。
type LicenseToken struct {
	Payload      string `json:"payload"`
	Signature    string `json:"signature"`
	RawPayload   []byte `json:"-"`
	RawSignature []byte `json:"-"`
}

// Encode 编码 Token 为 Base64 字符串。
func (t *LicenseToken) Encode() (string, error) {
	obj := map[string]string{
		"payload":   t.Payload,
		"signature": t.Signature,
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// DecodeToken 解码 Base64 编码的 Token。
func DecodeToken(encoded string) (*LicenseToken, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	var obj map[string]string
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal token: %w", err)
	}

	rawPayload, err := base64.StdEncoding.DecodeString(obj["payload"])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	rawSig, err := base64.StdEncoding.DecodeString(obj["signature"])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	return &LicenseToken{
		Payload:      obj["payload"],
		Signature:    obj["signature"],
		RawPayload:   rawPayload,
		RawSignature: rawSig,
	}, nil
}

// SignAndEncode 签名并编码 Token。
func SignAndEncode(privateKey ed25519.PrivateKey, payload Payload) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	sig, err := crypto.Sign(privateKey, data)
	if err != nil {
		return "", err
	}

	token := &LicenseToken{
		Payload:      base64.StdEncoding.EncodeToString(data),
		Signature:    base64.StdEncoding.EncodeToString(sig),
		RawPayload:   data,
		RawSignature: sig,
	}

	return token.Encode()
}

// VerifySignature 验证 Token 签名并解析 Payload。
func VerifySignature(publicKey ed25519.PublicKey, token *LicenseToken) (*Payload, error) {
	if err := crypto.Verify(publicKey, token.RawPayload, token.RawSignature); err != nil {
		return nil, fmt.Errorf("signature verification failed - token may be tampered")
	}

	var payload Payload
	if err := json.Unmarshal(token.RawPayload, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload")
	}

	return &payload, nil
}
