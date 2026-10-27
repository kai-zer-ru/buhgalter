package household

import "errors"

var (
	ErrAlreadyMember      = errors.New("already a household member")
	ErrNotMember          = errors.New("not a household member")
	ErrOwnerOnly          = errors.New("household owner only")
	ErrOwnerCannotLeave   = errors.New("owner cannot leave")
	ErrHasMembers         = errors.New("household has members")
	ErrHasSharedData      = errors.New("household has shared data")
	ErrNoOriginalOwner    = errors.New("unshare no original owner")
	ErrOwnerDeleteBlocked = errors.New("cannot delete household owner")
	ErrInviteInvalid      = errors.New("invite invalid")
	ErrMemberNotFound     = errors.New("household member not found")
	ErrCannotKickSelf     = errors.New("cannot kick self")
	ErrNameRequired       = errors.New("household name required")
	ErrNameTooLong        = errors.New("household name too long")
	ErrInvalidKind        = errors.New("household kind invalid")
)
