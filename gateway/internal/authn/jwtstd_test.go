package authn

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"maps"
	"reflect"
	"testing"
	"time"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authn"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	jwt "github.com/golang-jwt/jwt/v5"
)

func TestParseVerifyKey(t *testing.T) {
	t.Parallel()

	if got, err := ParseVerifyKey("HS256", []byte("secret")); err != nil || string(got.([]byte)) != "secret" {
		t.Errorf("HS256 passthrough = %v, %v; want raw bytes", got, err)
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	got, err := ParseVerifyKey("ES256", pemBytes)
	if err != nil {
		t.Fatalf("ES256 PEM parse failed: %v", err)
	}
	if !key.PublicKey.Equal(got.(*ecdsa.PublicKey)) {
		t.Error("parsed key does not match original")
	}

	if _, err := ParseVerifyKey("ES256", []byte("not pem")); err == nil {
		t.Error("expected error for non-PEM input")
	}
	if _, err := ParseVerifyKey("XX999", pemBytes); err == nil {
		t.Error("expected error for unknown signing method")
	}
}

func TestJWTAuthenticator(t *testing.T) {
	t.Parallel()

	keyES, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	keyHS := []byte("supersecretkey")

	keyRS, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	aES := NewJWTAuthenticator(&keyES.PublicKey, jwt.SigningMethodES256.Alg())
	aHS := NewJWTAuthenticator(keyHS, jwt.SigningMethodHS256.Alg())
	aRS := NewJWTAuthenticator(&keyRS.PublicKey, jwt.SigningMethodRS512.Alg())

	// Helper to build a signed token from a payload. additionalClaims may
	// override the defaults (e.g. "exp").
	makeToken := func(signingKey any, signingMethod jwt.SigningMethod, sub string, scope *resource.TokenScope, additionalClaims jwt.MapClaims) string {
		claims := jwt.MapClaims{
			"sub":   sub,
			"scope": scope,
			"exp":   time.Now().Add(time.Hour).Unix(),
		}
		maps.Copy(claims, additionalClaims)
		token := jwt.NewWithClaims(signingMethod, claims)
		s, err := token.SignedString(signingKey)
		if err != nil {
			t.Fatalf("failed to sign token with method %s: %v", signingMethod.Alg(), err)
			return ""
		}
		return s
	}

	wrongKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	tests := []struct {
		name          string
		token         string
		signingMethod jwt.SigningMethod
		wantSubject   string
		wantScope     resource.TokenScope
		wantErr       bool
	}{
		{
			name:          "valid credentials without scope",
			token:         makeToken(keyES, jwt.SigningMethodES256, "alice", nil, nil),
			signingMethod: jwt.SigningMethodES256,
			wantSubject:   "alice",
		},
		{
			name: "valid credentials with down-scope",
			token: makeToken(keyHS, jwt.SigningMethodHS256, "bob", &resource.TokenScope{
				Tenants:    []string{"t1"},
				Regions:    []string{"r1"},
				Workspaces: []string{"w1"},
			}, nil),
			signingMethod: jwt.SigningMethodHS256,
			wantSubject:   "bob",
			wantScope: resource.TokenScope{
				Tenants:    []string{"t1"},
				Regions:    []string{"r1"},
				Workspaces: []string{"w1"},
			},
		},
		{
			name:          "roles field in token is ignored",
			token:         makeToken(keyRS, jwt.SigningMethodRS512, "alice", nil, jwt.MapClaims{"roles": []string{"admin"}}),
			signingMethod: jwt.SigningMethodRS512,
			wantSubject:   "alice",
		},
		{
			name:          "signed with wrong key",
			token:         makeToken(wrongKey, jwt.SigningMethodES256, "alice", nil, nil),
			signingMethod: jwt.SigningMethodES256,
			wantErr:       true,
		},
		{
			name:          "not a JWT",
			token:         "this is not a JWT!!!",
			signingMethod: jwt.SigningMethodES256,
			wantErr:       true,
		},
		{
			name:          "expired token",
			token:         makeToken(keyES, jwt.SigningMethodES256, "alice", nil, jwt.MapClaims{"exp": time.Now().Add(-time.Hour).Unix()}),
			signingMethod: jwt.SigningMethodES256,
			wantErr:       true,
		},
		{
			name:          "missing username",
			token:         makeToken(keyES, jwt.SigningMethodES256, "", nil, nil),
			signingMethod: jwt.SigningMethodES256,
			wantErr:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			var id *authn.Identity
			var err error
			switch tc.signingMethod {
			case jwt.SigningMethodES256:
				id, err = aES.Authenticate(ctx, tc.token)
			case jwt.SigningMethodHS256:
				id, err = aHS.Authenticate(ctx, tc.token)
			case jwt.SigningMethodRS512:
				id, err = aRS.Authenticate(ctx, tc.token)
			default:
				t.Fatalf("unsupported signing method: %v", tc.signingMethod)
			}
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !isUnauthorized(err) {
					t.Errorf("expected ErrUnauthorized, got %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id.Subject != tc.wantSubject {
				t.Errorf("subject = %q, want %q", id.Subject, tc.wantSubject)
			}
			if !reflect.DeepEqual(id.TokenScope, tc.wantScope) {
				t.Errorf("token scope = %+v, want %+v", id.TokenScope, tc.wantScope)
			}
		})
	}
}
