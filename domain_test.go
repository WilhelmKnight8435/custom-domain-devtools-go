package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifySignatureAcceptsOnlyMatchingSecret(t *testing.T) {
	body := []byte(`{"event":"dns.domain.verification.completed","domain":"example.com"}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	good := hex.EncodeToString(mac.Sum(nil))
	if !verifySignature(body, good, "secret") {
		t.Fatal("matching signature rejected")
	}
	if verifySignature(body, "", "secret") {
		t.Fatal("empty signature accepted")
	}
}

func TestOnboardAddsRootDomainAndWritesFullHostname(t *testing.T) {
	var added, recordName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		switch r.URL.Path {
		case "/v1/dns/domain/add":
			added, _ = body["domain"].(string)
			w.Write([]byte(`{"ok":true,"data":{"zone_id":"zone-1"}}`))
		case "/v1/dns/record/upsert":
			recordName, _ = body["name"].(string)
			w.Write([]byte(`{"ok":true}`))
		case "/v1/account/webhooks/register":
			w.Write([]byte(`{"ok":true}`))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()
	c := &Client{BaseURL: server.URL, HTTP: server.Client()}
	if err := c.Onboard("app.example.co.uk", "target.example.com", "https://example.com/hook", "secret"); err != nil {
		t.Fatal(err)
	}
	if added != "example.co.uk" || recordName != "app.example.co.uk" {
		t.Fatalf("domain add = %q, record name = %q", added, recordName)
	}
}
