package iam

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePassword_ReturnsNonEmptyHashAndSalt(t *testing.T) {
	result, err := GeneratePassword("mysecretpassword")
	require.NoError(t, err)
	assert.NotEmpty(t, result.Hash)
	assert.NotEmpty(t, result.Salt)
}

func TestGeneratePassword_DifferentSaltsEachCall(t *testing.T) {
	r1, err := GeneratePassword("password")
	require.NoError(t, err)
	r2, err := GeneratePassword("password")
	require.NoError(t, err)

	assert.NotEqual(t, r1.Salt, r2.Salt, "each call must generate a unique random salt")
	assert.NotEqual(t, r1.Hash, r2.Hash, "different salts must produce different hashes")
}

func TestComparePassword_CorrectPassword_ReturnsTrue(t *testing.T) {
	password := "correct-horse-battery-staple"
	result, err := GeneratePassword(password)
	require.NoError(t, err)

	assert.True(t, ComparePassword(password, result.Hash, result.Salt))
}

func TestComparePassword_WrongPassword_ReturnsFalse(t *testing.T) {
	result, err := GeneratePassword("correct-password")
	require.NoError(t, err)

	assert.False(t, ComparePassword("wrong-password", result.Hash, result.Salt))
}

func TestComparePassword_WrongSalt_ReturnsFalse(t *testing.T) {
	password := "my-password"
	result, err := GeneratePassword(password)
	require.NoError(t, err)

	// Generate a different result to get a different (wrong) salt.
	other, err := GeneratePassword(password)
	require.NoError(t, err)

	assert.False(t, ComparePassword(password, result.Hash, other.Salt),
		"using the wrong salt must not verify the hash")
}
