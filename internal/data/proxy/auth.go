package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type bearerChallenge struct {
	Realm   string
	Service string
	Scope   string
}

//TODO: replace errors by Err const

func parseBearerChallenge(h string) (*bearerChallenge, error) {
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, errors.New("not a bearer challenge")
	}

	h = strings.TrimPrefix(h, "Bearer ")
	parts := strings.Split(h, ",")

	out := &bearerChallenge{}
	for _, p := range parts {
		kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
		if len(kv) != 2 {
			continue
		}
		v := strings.Trim(kv[1], `"`)

		switch kv[0] {
		case "realm":
			out.Realm = v
		case "service":
			out.Service = v
		case "scope":
			out.Scope = v
		}
	}

	if out.Realm == "" {
		return nil, errors.New("invalid bearer challenge")
	}
	return out, nil
}

func fetchBearerToken(proxy *Proxy, ch *bearerChallenge) (string, error) {
	req, err := http.NewRequest(http.MethodGet, ch.Realm, nil)
	if err != nil {
		return "", err
	}

	q := req.URL.Query()
	if ch.Service != "" {
		q.Set("service", ch.Service)
	}
	if ch.Scope != "" {
		q.Set("scope", ch.Scope)
	}
	req.URL.RawQuery = q.Encode()

	if proxy.Username != "" {
		req.SetBasicAuth(proxy.Username, proxy.Password)
	}

	//TODO: recreate or reuse the previous http client for upstream.

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed: %s", resp.Status)
	}

	var out struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}

	if out.Token != "" {
		return out.Token, nil
	}
	return out.AccessToken, nil
}
