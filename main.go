package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func verifySignature(body []byte, provided, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(provided))
}

func webhookHandler(secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !verifySignature(body, r.Header.Get("X-Infrai-Signature"), secret) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func main() {
	if len(os.Args) < 5 {
		log.Fatalf("usage: %s domain cname-target callback-url webhook-secret", os.Args[0])
	}
	c, err := NewClient()
	if err != nil {
		log.Fatal(err)
	}
	if err = c.Onboard(os.Args[1], os.Args[2], os.Args[3], os.Args[4]); err != nil {
		log.Fatal(err)
	}
	http.Handle("/webhooks/verification", webhookHandler(os.Args[4]))
	fmt.Println("domain onboarding submitted; waiting for verification webhook")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
