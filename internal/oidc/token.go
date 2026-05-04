// Podplane <https://podplane.dev>
// Copyright 2026 Nadrama Pty Ltd
// SPDX-License-Identifier: Apache-2.0

package oidc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// IsExpired returns true if the JWT id_token is within `skew` of expiring (or
// already expired). A malformed token is treated as expired.
func IsExpired(idToken string, skew time.Duration) bool {
	var claims struct {
		Exp json.Number `json:"exp"`
	}
	if err := decodeClaims(idToken, &claims); err != nil {
		return true
	}
	expSec, err := strconv.ParseInt(claims.Exp.String(), 10, 64)
	if err != nil {
		return true
	}
	return time.Now().Add(skew).After(time.Unix(expSec, 0))
}

// IdentityFromIDToken extracts the `sub` claim and the configured username
// claim (defaulting to "email") from an id_token. It does not verify the
// signature — verification already happened at the issuer when the token was
// minted.
func IdentityFromIDToken(idToken, usernameClaim string) (sub, email string, err error) {
	var raw map[string]any
	if err := decodeClaims(idToken, &raw); err != nil {
		return "", "", err
	}
	if v, ok := raw["sub"].(string); ok {
		sub = v
	}
	if v, ok := raw["email"].(string); ok {
		email = v
	}
	if usernameClaim != "" && usernameClaim != "email" {
		if v, ok := raw[usernameClaim].(string); ok && email == "" {
			email = v
		}
	}
	return sub, email, nil
}

// decodeClaims base64url-decodes the JWT payload (claims segment) of token
// into v.
func decodeClaims(token string, v any) error {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return fmt.Errorf("malformed jwt")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return fmt.Errorf("decode jwt payload: %w", err)
		}
	}
	if err := json.Unmarshal(payload, v); err != nil {
		return fmt.Errorf("parse jwt claims: %w", err)
	}
	return nil
}
