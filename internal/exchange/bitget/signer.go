package bitget

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

type Signer struct {
	secret []byte
}

func NewSigner(secret string) *Signer {
	return &Signer{secret: []byte(secret)}
}

func (signer *Signer) Sign(ts, method, path, body string) string {
	message := ts + strings.ToUpper(method) + path + body
	mac := hmac.New(sha256.New, signer.secret)
	_, _ = mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
