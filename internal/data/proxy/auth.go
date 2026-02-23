// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type BearerChallenge struct {
	Realm   string
	Service string
	Scope   string
}

//TODO: replace errors by Err const

func ParseBearerChallenge(h string) (*BearerChallenge, error) {
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, errors.New("not a bearer challenge")
	}

	h = strings.TrimPrefix(h, "Bearer ")
	parts := strings.Split(h, ",")

	out := &BearerChallenge{}
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

func FetchBearerToken(proxy *Proxy, ch *BearerChallenge) (string, error) {
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
