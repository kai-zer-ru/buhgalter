package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserStatusValid(t *testing.T) {
	require.True(t, UserStatusActive.Valid())
	require.True(t, UserStatusPending.Valid())
	require.True(t, UserStatusBanned.Valid())
	require.False(t, UserStatus("unknown").Valid())
}

func TestIsActive(t *testing.T) {
	require.True(t, IsActive("active"))
	require.False(t, IsActive("pending"))
	require.False(t, IsActive("banned"))
}

func TestCanTransition(t *testing.T) {
	require.True(t, CanTransition(UserStatusPending, UserStatusActive))
	require.True(t, CanTransition(UserStatusPending, UserStatusBanned))
	require.True(t, CanTransition(UserStatusActive, UserStatusBanned))
	require.True(t, CanTransition(UserStatusBanned, UserStatusActive))

	require.False(t, CanTransition(UserStatusActive, UserStatusPending))
	require.False(t, CanTransition(UserStatusBanned, UserStatusPending))
	require.False(t, CanTransition(UserStatusActive, UserStatusActive))
	require.False(t, CanTransition(UserStatusPending, UserStatusPending))
}
