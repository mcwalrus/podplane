// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package oidcserver

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// Handler returns an http.Handler implementing a minimal local OIDC provider.
//
// The handler is intended to be mounted at the root of issuerURL (the caller
// is responsible for stripping any path prefix). It serves:
//
//	GET  /.well-known/openid-configuration
//	GET  /.well-known/jwks.json
//	GET  /authorize  — auto-approves and 302-redirects with `code`
//	POST /token      — handles authorization_code and refresh_token grants
//
// Tokens are signed with the supplied RSA private key and contain hard-coded
// "test-user" identity claims; this is strictly for local development.
// validateClientID is called with client_id before a token is issued.
func Handler(issuerURL string, key *rsa.PrivateKey, validateClientID func(clientID string) error) (http.Handler, error) {
	if validateClientID == nil {
		return nil, fmt.Errorf("validate client ID callback is required")
	}
	jwks, err := buildJWKS(key)
	if err != nil {
		return nil, err
	}
	codes := newCodeStore()

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", discoveryHandler(issuerURL))
	mux.HandleFunc("/.well-known/jwks.json", jwksHandler(jwks))
	mux.HandleFunc("/authorize", authorizeHandler(codes))
	mux.HandleFunc("/token", tokenHandler(codes, key, issuerURL, validateClientID))
	return mux, nil
}

// buildJWKS wraps the public half of key in a single-key JWKS suitable for
// the /.well-known/jwks.json endpoint.
func buildJWKS(key *rsa.PrivateKey) (jwk.Set, error) {
	pub, err := jwk.FromRaw(key.Public())
	if err != nil {
		return nil, fmt.Errorf("oidc: build public JWK: %w", err)
	}
	for k, v := range map[string]any{
		jwk.KeyIDKey:     "podplane-local",
		jwk.AlgorithmKey: jwa.RS256,
		jwk.KeyUsageKey:  "sig",
	} {
		if err := pub.Set(k, v); err != nil {
			return nil, err
		}
	}
	set := jwk.NewSet()
	if err := set.AddKey(pub); err != nil {
		return nil, err
	}
	return set, nil
}

// writeJSON encodes v as pretty JSON to w with the right Content-Type.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}
