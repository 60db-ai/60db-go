package sixtydb

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"strconv"
	"time"
)

// Webhook header names.
const (
	WebhookTimestampHeader = "X-60db-Timestamp"
	WebhookSignatureHeader = "X-60db-Signature"
)

// ComputeWebhookSignature returns "v1=" + hex(HMAC-SHA256(secret, "{timestamp}.{rawBody}")).
func ComputeWebhookSignature(rawBody []byte, timestamp, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(timestamp + "."))
	m.Write(rawBody)
	return "v1=" + hex.EncodeToString(m.Sum(nil))
}

// VerifyWebhookSignature checks a signature in constant time with replay protection.
// toleranceSec <= 0 means the default 300 s. nowSec <= 0 means the current time.
func VerifyWebhookSignature(rawBody []byte, signature, timestamp, secret string, toleranceSec, nowSec int64) bool {
	if secret == "" || signature == "" || len(timestamp) < 1 || len(timestamp) > 12 {
		return false
	}
	for _, r := range timestamp {
		if r < '0' || r > '9' {
			return false
		}
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false
	}
	if toleranceSec <= 0 {
		toleranceSec = 300
	}
	if nowSec <= 0 {
		nowSec = time.Now().Unix()
	}
	if math.Abs(float64(nowSec-ts)) > float64(toleranceSec) {
		return false
	}
	expected := ComputeWebhookSignature(rawBody, timestamp, secret)
	return hmac.Equal([]byte(expected), []byte(signature))
}
