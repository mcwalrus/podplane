// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package seeds

import "testing"

func TestParseNameDefaultsAndValidates(t *testing.T) {
	cases := map[string]string{
		"":          Recommended,
		Recommended: Recommended,
		Minimal:     Minimal,
		None:        None,
	}
	for input, want := range cases {
		got, err := ParseName(input)
		if err != nil {
			t.Fatalf("ParseName(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("ParseName(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := ParseName("bogus"); err == nil {
		t.Fatalf("expected invalid seed name error")
	}
}

func TestResolveSeedPathNoneSkips(t *testing.T) {
	path, err := ResolveSeedPath(ResolveOptions{Name: None})
	if err != nil {
		t.Fatalf("ResolveSeedPath error = %v", err)
	}
	if path != "" {
		t.Fatalf("ResolveSeedPath path = %q, want empty", path)
	}
}
