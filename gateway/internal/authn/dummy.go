// Package authn provides authentication implementations for the ECP gateway.
package authn

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authnport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
)

// tokenPayload is the expected JSON structure of a Dummy bearer token.
// The token is a standard base64-encoded JSON object (no padding normalization needed
// since the SDK client encodes it verbatim).
type tokenPayload struct {
	Username string   `json:"username"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

// DummyAuthenticator validates bearer tokens using a static user→password map.
//
// WARNING: this authenticator is for development and testing ONLY.
// There is no cryptographic signature; clients self-assert their roles.
// Do NOT use in production.
//
// Token format: base64(JSON{"username":"alice","password":"s3cr3t","roles":["role1","role2"]})
type DummyAuthenticator struct {
	// users maps username → expected password.
	users map[string]string
}

// NewDummyAuthenticator creates a DummyAuthenticator with the given credential map.
// The map keys are usernames; the values are the expected plain-text passwords.
func NewDummyAuthenticator(users map[string]string) *DummyAuthenticator {
	return &DummyAuthenticator{users: users}
}

// Authenticate implements authnport.Authenticator.
// It decodes the base64 token, parses the JSON payload, and validates the
// username+password pair against the configured map.
// Returns kernel.ErrUnauthorized when the token is malformed or credentials are invalid.
func (d *DummyAuthenticator) Authenticate(_ context.Context, token string) (*authnport.Identity, error) {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		// Try URL-safe base64 as a fallback (some HTTP clients use it).
		raw, err = base64.URLEncoding.DecodeString(token)
		if err != nil {
			return nil, fmt.Errorf("%w: token is not valid base64", kernel.ErrUnauthorized)
		}
	}

	var payload tokenPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("%w: token payload is not valid JSON", kernel.ErrUnauthorized)
	}

	if payload.Username == "" {
		return nil, fmt.Errorf("%w: token missing username", kernel.ErrUnauthorized)
	}

	expected, ok := d.users[payload.Username]
	if !ok || expected != payload.Password {
		return nil, fmt.Errorf("%w: invalid credentials", kernel.ErrUnauthorized)
	}

	if payload.Roles == nil {
		payload.Roles = []string{}
	}

	return &authnport.Identity{
		Subject: payload.Username,
		Roles:   payload.Roles,
	}, nil
}
