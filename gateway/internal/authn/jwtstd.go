// Package authn provides JWT authentication implementations for the ECP gateway.
package authn

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authnport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	jwt "github.com/golang-jwt/jwt/v5"
)

// ParseVerifyKey turns the contents of a key file into the verification key
// golang-jwt expects for the given signing method: the raw bytes for HS*
// (they are the HMAC secret), or the PEM-encoded PKIX public key
// (RSA, ECDSA, Ed25519) for every other method.
func ParseVerifyKey(method string, data []byte) (any, error) {
	if jwt.GetSigningMethod(method) == nil {
		return nil, fmt.Errorf("unknown JWT signing method %q", method)
	}
	if strings.HasPrefix(method, "HS") {
		return data, nil
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("JWT key is not PEM-encoded")
	}
	return x509.ParsePKIXPublicKey(block.Bytes)
}

// JwtAuthenticator validates standard signed JWT bearer tokens
// (the raw "header.payload.signature" compact form, no extra encoding).
// The key type must match the signing method: []byte for HS*,
// *ecdsa.PublicKey for ES*, *rsa.PublicKey for RS*/PS*, ed25519.PublicKey
// for EdDSA — see [ParseVerifyKey].
type JwtAuthenticator struct {
	secret        any
	signingMethod string
}

// NewJWTAuthenticator creates a JwtAuthenticator.
func NewJWTAuthenticator(secret any, signingMethod string) *JwtAuthenticator {
	return &JwtAuthenticator{
		secret:        secret,
		signingMethod: signingMethod,
	}
}

// jwtClaims is the expected JWT payload. Embedding RegisteredClaims provides
// exp/sub validation; the optional "scope" object unmarshals directly into
// [resource.TokenScope] and down-scopes the caller's permissions.
type jwtClaims struct {
	jwt.RegisteredClaims
	Scope *resource.TokenScope `json:"scope,omitempty"`
}

// Authenticate implements authnport.Authenticator, verifies the JWT token, and returns an Identity carrying the subject and any optional down-scoping asserted by the token.
// Returns kernel.ErrUnauthorized when the token is malformed or credentials are invalid.
func (j *JwtAuthenticator) Authenticate(_ context.Context, tokenString string) (*authnport.Identity, error) {
	claims := &jwtClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return j.secret, nil
	}, jwt.WithValidMethods([]string{j.signingMethod}), jwt.WithExpirationRequired())
	if err != nil {
<<<<<<< HEAD
		return nil, fmt.Errorf("%w: token is not valid JWT: %w", kernel.ErrUnauthorized, err)
=======
		return nil, fmt.Errorf("%w: token is not valid JWT: %v", kernel.ErrUnauthorized, err)
>>>>>>> 6768168d (feat: add jwt authentication)
	}

	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: token subject is missing", kernel.ErrUnauthorized)
	}

	scope := resource.TokenScope{}
	if claims.Scope != nil {
		scope = *claims.Scope
	}
	return &authnport.Identity{
		Subject:    claims.Subject,
		TokenScope: scope,
	}, nil
}
