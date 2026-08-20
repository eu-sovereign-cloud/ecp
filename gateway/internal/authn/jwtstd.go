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
//
// The verification key, the accepted signing method and the expected issuer and
// audience all come from the operator's configuration; nothing the token itself names
// is trusted to select them.
type JwtAuthenticator struct {
	secret     any
	parserOpts []jwt.ParserOption
}

// NewJWTAuthenticator creates a JwtAuthenticator.
//
// issuer and audience are the expected "iss" and "aud" claim values. Each is enforced
// only when non-empty, and enforcement also makes the claim mandatory: a token that
// omits a configured claim is rejected. Leave them empty to accept any issuer/audience
// (the pre-existing behaviour) — appropriate only when a single issuer mints tokens for
// this gateway alone.
func NewJWTAuthenticator(secret any, signingMethod, issuer, audience string) *JwtAuthenticator {
	opts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{signingMethod}),
		jwt.WithExpirationRequired(),
	}
	if issuer != "" {
		opts = append(opts, jwt.WithIssuer(issuer))
	}
	if audience != "" {
		opts = append(opts, jwt.WithAudience(audience))
	}
	return &JwtAuthenticator{
		secret:     secret,
		parserOpts: opts,
	}
}

// jwtClaims is the expected JWT payload. Embedding RegisteredClaims provides
// exp/sub/iss/aud validation; the optional "scope" object unmarshals directly into
// [resource.TokenScope] and down-scopes the caller's permissions, while the optional
// "tenants" claim carries the issuer-asserted tenant membership.
type jwtClaims struct {
	jwt.RegisteredClaims
	Scope   *resource.TokenScope `json:"scope,omitempty"`
	Tenants []string             `json:"tenants,omitempty"`
}

// Authenticate implements authnport.Authenticator, verifies the JWT token, and returns an Identity carrying the subject, the issuer-asserted tenant membership and any optional down-scoping asserted by the token.
// Returns kernel.ErrUnauthorized when the token is malformed, expired, signed by an unexpected key or method, or carries the wrong issuer or audience.
func (j *JwtAuthenticator) Authenticate(_ context.Context, tokenString string) (*authnport.Identity, error) {
	claims := &jwtClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return j.secret, nil
	}, j.parserOpts...)
	if err != nil {
		return nil, fmt.Errorf("%w: token is not valid JWT: %w", kernel.ErrUnauthorized, err)
	}

	if claims.Subject == "" {
		return nil, fmt.Errorf("%w: token subject is missing", kernel.ErrUnauthorized)
	}

	scope := resource.TokenScope{}
	if claims.Scope != nil {
		scope = *claims.Scope
	}
	return &authnport.Identity{
		Subject:       claims.Subject,
		TokenScope:    scope,
		MemberTenants: claims.Tenants,
	}, nil
}
