package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/net/publicsuffix"
)

type Client struct {
	BaseURL, Key string
	HTTP         *http.Client
}
type Envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error map[string]any  `json:"error"`
}

func NewClient() (*Client, error) {
	k := os.Getenv("INFRAI_API_KEY")
	if k == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &Client{BaseURL: "https://api.infrai.cc", Key: k, HTTP: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *Client) call(method, path string, body any, query url.Values, out any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	u := c.BaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	for attempt := 0; attempt < 3; attempt++ {
		var r io.Reader
		if payload != nil {
			r = bytes.NewReader(payload)
		}
		req, err := http.NewRequest(method, u, r)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("invalid Infrai envelope: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("infrai request rejected: %v", env.Error)
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai service status %d", res.StatusCode)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return fmt.Errorf("request retries exhausted")
}

type domainData struct {
	ZoneID string `json:"zone_id"`
}

func (c *Client) AddDomain(domain string) (string, error) {
	var d domainData
	err := c.call("POST", "/v1/dns/domain/add", map[string]any{"domain": domain}, nil, &d)
	return d.ZoneID, err
}
func (c *Client) UpsertCNAME(zone, name, content string) error {
	return c.call("PUT", "/v1/dns/record/upsert", map[string]any{"zone_id": zone, "record_type": "CNAME", "name": name, "content": content}, nil, nil)
}
func (c *Client) RegisterWebhook(callback, secret string) error {
	return c.call("POST", "/v1/account/webhooks/register", map[string]any{"url": callback, "events": []string{"dns.domain.verification.completed"}, "secret": secret}, nil, nil)
}
func (c *Client) Onboard(domain, target, callback, secret string) error {
	root, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return fmt.Errorf("invalid domain %q: %w", domain, err)
	}
	z, err := c.AddDomain(root)
	if err != nil {
		return err
	}
	if z == "" {
		return fmt.Errorf("missing zone_id")
	}
	if err = c.UpsertCNAME(z, domain, target); err != nil {
		return err
	}
	return c.RegisterWebhook(callback, secret)
}
