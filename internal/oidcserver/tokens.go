// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package oidcserver

import (
	"crypto/rsa"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// idTokenTTL is how long a freshly-minted id_token (and its mirroring
// access_token) is valid for.
const idTokenTTL = time.Hour

// LocalSub is the `sub` claim baked into every token the local fake OIDC
// issues. Exported so callers (e.g. `podplane local start` configuring
// kubectl) can build deterministic user names without performing a login
// first.
const LocalSub = "test-user"

// IssueLocalToken signs and returns a fresh id_token suitable for the local
// fake OIDC. Identity claims are hard-coded to LocalSub / "test@localhost" with
// the system:masters group — this is a local development fixture only.
// clusterID becomes the audience and must match the apiserver's configured
// --oidc-client-id; issuerURL must match its --oidc-issuer-url.
func IssueLocalToken(key *rsa.PrivateKey, issuerURL, clusterID string) (string, error) {
	if clusterID == "" {
		clusterID = "podplane-local"
	}
	now := time.Now()
	tok, err := jwt.NewBuilder().
		Issuer(issuerURL).
		Audience([]string{clusterID}).
		Subject(LocalSub).
		IssuedAt(now).
		Expiration(now.Add(idTokenTTL)).
		Claim("email", "test@localhost").
		Claim("groups", []string{"system:masters"}).
		Build()
	if err != nil {
		return "", fmt.Errorf("build token: %w", err)
	}
	headers := jws.NewHeaders()
	_ = headers.Set(jws.KeyIDKey, "podplane-local")
	_ = headers.Set(jws.TypeKey, "JWT")
	signed, err := jwt.NewSerializer().
		Sign(jwt.WithKey(jwa.RS256, key, jws.WithProtectedHeaders(headers))).
		Serialize(tok)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return string(signed), nil
}

// issueToken signs a fresh id_token and writes the standard token response.
// Local cluster configs set OIDC client_id to the cluster ID, which is also
// what kube-apiserver expects in the token audience.
func issueToken(w http.ResponseWriter, key *rsa.PrivateKey, issuerURL, clusterID string) error {
	signed, err := IssueLocalToken(key, issuerURL, clusterID)
	if err != nil {
		return err
	}
	refresh, err := randomHex(32)
	if err != nil {
		return fmt.Errorf("generate refresh token: %w", err)
	}
	writeJSON(w, map[string]any{
		"access_token":  signed,
		"id_token":      signed,
		"token_type":    "Bearer",
		"expires_in":    int(idTokenTTL.Seconds()),
		"refresh_token": refresh,
	})
	return nil
}
