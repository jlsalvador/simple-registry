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

package rbac

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	netHttp "net/http"
	"regexp"
	"strings"
	"time"
)

const AnonymousUsername = "anonymous"

var httpAuthBasicRegexp = regexp.MustCompile(`^Basic\s+([a-zA-Z0-9+/]+={0,2})$`)
var httpAuthBearerRegexp = regexp.MustCompile(`^Bearer\s+([a-zA-Z0-9._+/=-]+)$`)

func (e *Engine) isBasicAuthAllowed(
	r *netHttp.Request,
	resource,
	scope,
	verb string,
) bool {
	rUsr, rPwd, ok := r.BasicAuth()

	// Check if the user exists and password is valid.
	if !ok || !e.HasUser(rUsr, rPwd) {
		return false
	}

	// User is validated, check if it's allowed to perform the action.
	return e.IsAllowed(rUsr, resource, scope, verb)
}

func (e *Engine) isBearerAllowed(
	r *netHttp.Request,
	resource,
	scope,
	verb string,
) bool {
	// Extract the token from the Authorization header.
	auth := r.Header.Get("Authorization")
	matches := httpAuthBearerRegexp.FindStringSubmatch(auth)
	if len(matches) != 2 {
		return false
	}
	// The token is now a standard JWT string "header.payload.signature".
	token := matches[1]

	// Split the JWT into its three parts.
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}
	header := parts[0]
	payload := parts[1]
	signatureReceived := parts[2]

	// Verify the HMAC-SHA512 signature.
	// We sign the "header.payload" part exactly as it was received.
	dataToVerify := []byte(header + "." + payload)
	h := hmac.New(sha512.New, e.TokenSecret)
	h.Write(dataToVerify)
	expectedSig := h.Sum(nil)

	// We use RawURLEncoding to match the standard JWT format (no padding).
	expectedSigB64 := base64.RawURLEncoding.EncodeToString(expectedSig)

	// Constant-time comparison to prevent timing attacks.
	if !hmac.Equal([]byte(signatureReceived), []byte(expectedSigB64)) {
		return false
	}

	// Decode the payload.
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return false
	}
	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return false
	}

	// Validate token expiration (iat + timeout).
	if iat, ok := claims["iat"].(float64); ok {
		issuedAt := time.Unix(int64(iat), 0)
		since := time.Since(issuedAt)
		if since > e.TokenTimeout {
			return false // Token expired.
		}
	} else {
		return false // Missing or invalid iat.
	}

	// Final RBAC check.
	username, ok := claims["sub"].(string)
	if !ok {
		return false
	}
	return e.IsAllowed(username, resource, scope, verb)
}

func (e *Engine) IsRequestAllowed(
	r *netHttp.Request,
	resource string,
	scope string,
	verb string,
) bool {
	auth := r.Header.Get("Authorization")

	// Basic auth.
	if httpAuthBasicRegexp.MatchString(auth) {
		return e.isBasicAuthAllowed(r, resource, scope, verb)
	}

	// Bearer token auth.
	if httpAuthBearerRegexp.MatchString(auth) {
		return e.isBearerAllowed(r, resource, scope, verb)
	}

	// Anonymous auth.
	if e.IsAnonymousUserEnabled() {
		return e.IsAllowed(AnonymousUsername, resource, scope, verb)
	}

	return false
}

// GenerateToken creates a standard, URL-safe JWT token
func (e *Engine) GenerateToken(user, scope string) (string, error) {
	// Define the JWT header.
	headerJSON := `{"alg":"HS512","typ":"JWT"}`
	header := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))

	// Define the JWT payload.
	payloadMap := map[string]any{
		"sub":   user,
		"scope": scope,
		"iat":   time.Now().Unix(),
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// Construct the signing input (header.payload).
	signingInput := header + "." + payload

	// Generate the signature.
	h := hmac.New(sha512.New, e.TokenSecret)
	h.Write([]byte(signingInput))
	signature := h.Sum(nil)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	// Combine all parts into a standard JWT string "header.payload.signature".
	return signingInput + "." + signatureB64, nil
}
