// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package oidcserver

import (
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"time"
)

// discoveryHandler serves /.well-known/openid-configuration.
func discoveryHandler(issuerURL string) http.HandlerFunc {
	doc := map[string]any{
		"issuer":                                issuerURL,
		"jwks_uri":                              issuerURL + "/.well-known/jwks.json",
		"authorization_endpoint":                issuerURL + "/authorize",
		"token_endpoint":                        issuerURL + "/token",
		"response_types_supported":              []string{"code", "id_token", "token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, doc)
	}
}

// jwksHandler serves /.well-known/jwks.json.
func jwksHandler(jwks any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, jwks)
	}
}

// authorizeHandler serves /authorize. It auto-approves the request, stores
// the PKCE challenge, and 302-redirects back to the relying party with a
// fresh authorization code.
func authorizeHandler(codes *codeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		q := r.URL.Query()
		redirect := q.Get("redirect_uri")
		if redirect == "" {
			http.Error(w, "missing redirect_uri", http.StatusBadRequest)
			return
		}
		dest, err := url.Parse(redirect)
		if err != nil {
			http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
			return
		}
		code, err := randomHex(32)
		if err != nil {
			http.Error(w, "generate code: "+err.Error(), http.StatusInternalServerError)
			return
		}
		codes.put(code, &authCode{
			ClientID:        q.Get("client_id"),
			RedirectURI:     redirect,
			CodeChallenge:   q.Get("code_challenge"),
			ChallengeMethod: q.Get("code_challenge_method"),
			ExpiresAt:       time.Now().Add(authCodeTTL),
		})
		dq := dest.Query()
		dq.Set("code", code)
		if state := q.Get("state"); state != "" {
			dq.Set("state", state)
		}
		dest.RawQuery = dq.Encode()
		http.Redirect(w, r, dest.String(), http.StatusFound)
	}
}

// tokenHandler serves /token. It branches on grant_type.
func tokenHandler(codes *codeStore, key *rsa.PrivateKey, issuerURL string, validateClientID func(clientID string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
			return
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			if !verifyAuthCode(w, r, codes) {
				return
			}
		case "refresh_token":
			if r.Form.Get("refresh_token") == "" {
				http.Error(w, "missing refresh_token", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "unsupported grant_type", http.StatusBadRequest)
			return
		}
		clientID := r.Form.Get("client_id")
		if err := validateClientID(clientID); err != nil {
			http.Error(w, "invalid client_id: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := issueToken(w, key, issuerURL, clientID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// verifyAuthCode validates the code + PKCE verifier from a /token request.
// Writes a 4xx and returns false on any failure.
func verifyAuthCode(w http.ResponseWriter, r *http.Request, codes *codeStore) bool {
	code := r.Form.Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return false
	}
	stored := codes.take(code)
	if stored == nil {
		http.Error(w, "invalid or expired code", http.StatusBadRequest)
		return false
	}
	if r.Form.Get("client_id") != stored.ClientID {
		http.Error(w, "client_id mismatch", http.StatusBadRequest)
		return false
	}
	if r.Form.Get("redirect_uri") != stored.RedirectURI {
		http.Error(w, "redirect_uri mismatch", http.StatusBadRequest)
		return false
	}
	if stored.CodeChallenge == "" {
		return true // no PKCE required
	}
	verifier := r.Form.Get("code_verifier")
	if verifier == "" {
		http.Error(w, "missing code_verifier", http.StatusBadRequest)
		return false
	}
	method := stored.ChallengeMethod
	if method == "" {
		method = "plain"
	}
	var got string
	switch method {
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		got = base64.RawURLEncoding.EncodeToString(sum[:])
	case "plain":
		got = verifier
	default:
		http.Error(w, "unsupported code_challenge_method", http.StatusBadRequest)
		return false
	}
	if got != stored.CodeChallenge {
		http.Error(w, "code_verifier mismatch", http.StatusBadRequest)
		return false
	}
	return true
}
