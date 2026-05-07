package iam

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testJWTInstance() *JWT {
	return NewJWT(JWTConfig{
		Secret:    "test-secret-32-bytes-minimum-len",
		Expire:    time.Minute,
		Algorithm: "HS256",
	})
}

func TestGenerateToken_NonEmpty(t *testing.T) {
	j := testJWTInstance()
	tok, err := j.GenerateToken("user-public-id")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)
}

func TestGenerateToken_DifferentJTIEachCall(t *testing.T) {
	j := testJWTInstance()
	tok1, err := j.GenerateToken("uid-1")
	require.NoError(t, err)
	tok2, err := j.GenerateToken("uid-1")
	require.NoError(t, err)

	claims1, err := j.ParseToken(tok1)
	require.NoError(t, err)
	claims2, err := j.ParseToken(tok2)
	require.NoError(t, err)

	assert.NotEqual(t, claims1.JTI, claims2.JTI, "two calls must produce different JTI values")
}

func TestGenerateToken_ParseableByIssuer(t *testing.T) {
	j := testJWTInstance()
	tok, err := j.GenerateToken("my-public-id")
	require.NoError(t, err)

	claims, err := j.ParseToken(tok)
	require.NoError(t, err)
	assert.Equal(t, "my-public-id", claims.PublicID)
	assert.NotEmpty(t, claims.JTI)
	assert.False(t, claims.ExpiresAt.IsZero())
}

func TestParseToken_HappyPath(t *testing.T) {
	j := testJWTInstance()
	publicID := "abc-def-ghi"
	tok, err := j.GenerateToken(publicID)
	require.NoError(t, err)

	claims, err := j.ParseToken(tok)
	require.NoError(t, err)
	assert.Equal(t, publicID, claims.PublicID)
	assert.NotEmpty(t, claims.JTI)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestParseToken_ExpiredToken(t *testing.T) {
	j := NewJWT(JWTConfig{
		Secret:    "test-secret-32-bytes-minimum-len",
		Expire:    -time.Second, // already expired
		Algorithm: "HS256",
	})
	tok, err := j.GenerateToken("uid")
	require.NoError(t, err)

	_, err = j.ParseToken(tok)
	assert.Error(t, err)
}

func TestParseToken_WrongAlgorithm(t *testing.T) {
	// Sign with HS512, parse with HS256 instance — must be rejected
	signer := NewJWT(JWTConfig{
		Secret:    "test-secret-32-bytes-minimum-len",
		Expire:    time.Minute,
		Algorithm: "HS512",
	})
	parser := NewJWT(JWTConfig{
		Secret:    "test-secret-32-bytes-minimum-len",
		Expire:    time.Minute,
		Algorithm: "HS256",
	})

	tok, err := signer.GenerateToken("uid")
	require.NoError(t, err)

	_, err = parser.ParseToken(tok)
	assert.Error(t, err)
}

func TestParseToken_MissingJTI(t *testing.T) {
	// Hand-craft a valid HS256 token without a jti claim.
	secret := []byte("test-secret-32-bytes-minimum-len")
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "uid",
		"iat": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
		// jti deliberately omitted
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	require.NoError(t, err)

	j := testJWTInstance()
	_, err = j.ParseToken(signed)
	assert.Error(t, err)
}

func TestParseToken_EmptyJTI(t *testing.T) {
	// Hand-craft a valid HS256 token with jti == "".
	secret := []byte("test-secret-32-bytes-minimum-len")
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "uid",
		"jti": "",
		"iat": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	require.NoError(t, err)

	j := testJWTInstance()
	_, err = j.ParseToken(signed)
	assert.Error(t, err)
}

func TestParseToken_MissingSub(t *testing.T) {
	// Hand-craft a valid HS256 token with no sub claim.
	secret := []byte("test-secret-32-bytes-minimum-len")
	now := time.Now()
	claims := jwt.MapClaims{
		"jti": "some-jti",
		"iat": now.Unix(),
		"exp": now.Add(time.Minute).Unix(),
		// sub deliberately omitted
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(secret)
	require.NoError(t, err)

	j := testJWTInstance()
	_, err = j.ParseToken(signed)
	assert.Error(t, err)
}

func TestParseToken_TamperedSignature(t *testing.T) {
	j := testJWTInstance()
	tok, err := j.GenerateToken("uid")
	require.NoError(t, err)

	// Corrupt the last few characters of the signature.
	tampered := tok[:len(tok)-4] + "XXXX"
	_, err = j.ParseToken(tampered)
	assert.Error(t, err)
}

func TestParseToken_WrongIssuer(t *testing.T) {
	signer := NewJWT(JWTConfig{
		Secret:    "test-secret-32-bytes-minimum-len",
		Expire:    time.Minute,
		Algorithm: "HS256",
		Issuer:    "issuer-a",
	})
	parser := NewJWT(JWTConfig{
		Secret:    "test-secret-32-bytes-minimum-len",
		Expire:    time.Minute,
		Algorithm: "HS256",
		Issuer:    "issuer-b",
	})

	tok, err := signer.GenerateToken("uid")
	require.NoError(t, err)

	_, err = parser.ParseToken(tok)
	assert.Error(t, err)
}

func TestParseToken_WrongAudience(t *testing.T) {
	signer := NewJWT(JWTConfig{
		Secret:    "test-secret-32-bytes-minimum-len",
		Expire:    time.Minute,
		Algorithm: "HS256",
		Audience:  "audience-a",
	})
	parser := NewJWT(JWTConfig{
		Secret:    "test-secret-32-bytes-minimum-len",
		Expire:    time.Minute,
		Algorithm: "HS256",
		Audience:  "audience-b",
	})

	tok, err := signer.GenerateToken("uid")
	require.NoError(t, err)

	_, err = parser.ParseToken(tok)
	assert.Error(t, err)
}
