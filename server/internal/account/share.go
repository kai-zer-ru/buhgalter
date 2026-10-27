package account

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/auth"
	"github.com/kai-zer-ru/buhgalter/internal/balancehooks"
	"github.com/kai-zer-ru/buhgalter/internal/household"
	"github.com/kai-zer-ru/buhgalter/internal/realtime"
)

type ShareResult struct {
	Account
	Details ShareDetails `json:"details"`
}

type ShareDetails struct {
	Moved household.MovedCounts `json:"moved"`
}

func Share(ctx context.Context, db *sql.DB, sc auth.Scope, id string) (ShareResult, error) {
	if !sc.InHousehold() {
		return ShareResult{}, ErrNotHouseholdMember
	}
	acc, err := GetByScope(ctx, db, sc, id)
	if err != nil {
		return ShareResult{}, err
	}
	if acc.OwnerUserID != sc.UserID {
		return ShareResult{}, ErrNotFound
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ShareResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	from := sc.UserID
	moved, err := household.MoveAccount(ctx, tx, id, sc.UserID, sc.PrincipalID, household.MoveOpts{OriginalOwner: &from})
	if errors.Is(err, sql.ErrNoRows) {
		return ShareResult{}, ErrNotFound
	}
	if err != nil {
		return ShareResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ShareResult{}, err
	}
	_ = accountbalance.Refresh(ctx, db, sc.UserID, id)
	_ = accountbalance.Refresh(ctx, db, sc.PrincipalID, id)
	balancehooks.NotifyRefresh(ctx, db, sc.PrincipalID, time.Time{}, id)
	realtime.NotifyOwnerDataChanged(ctx, db, sc.PrincipalID)
	out, err := GetByScope(ctx, db, sc, id)
	if err != nil {
		return ShareResult{}, err
	}
	return ShareResult{Account: out, Details: ShareDetails{Moved: moved}}, nil
}

func Unshare(ctx context.Context, db *sql.DB, sc auth.Scope, id string) (ShareResult, error) {
	if !sc.InHousehold() {
		return ShareResult{}, ErrNotHouseholdMember
	}
	if !sc.IsOwner() {
		return ShareResult{}, household.ErrOwnerOnly
	}
	acc, err := GetByScope(ctx, db, sc, id)
	if err != nil {
		return ShareResult{}, err
	}
	if acc.OwnerUserID != sc.PrincipalID {
		return ShareResult{}, ErrNotFound
	}
	if acc.OriginalOwner == nil || acc.OriginalOwner.UserID == "" {
		return ShareResult{}, household.ErrNoOriginalOwner
	}
	orig := acc.OriginalOwner.UserID
	if _, err := household.GetMember(ctx, db, sc.HouseholdID, orig); err != nil {
		return ShareResult{}, household.ErrNoOriginalOwner
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ShareResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	moved, err := household.MoveAccount(ctx, tx, id, sc.PrincipalID, orig, household.MoveOpts{})
	if errors.Is(err, sql.ErrNoRows) {
		return ShareResult{}, ErrNotFound
	}
	if err != nil {
		return ShareResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ShareResult{}, err
	}
	_ = accountbalance.Refresh(ctx, db, orig, id)
	_ = accountbalance.Refresh(ctx, db, sc.PrincipalID, id)
	balancehooks.NotifyRefresh(ctx, db, orig, time.Time{}, id)
	realtime.NotifyOwnerDataChanged(ctx, db, sc.PrincipalID)
	realtime.NotifyOwnerDataChanged(ctx, db, orig)
	out, err := GetByScope(ctx, db, auth.Scope{UserID: orig}, id)
	if err != nil {
		return ShareResult{}, err
	}
	return ShareResult{Account: out, Details: ShareDetails{Moved: moved}}, nil
}

func IsLinkedPersonal(err error) bool {
	return errors.Is(err, household.ErrLinkedPersonal)
}

func LinkedPersonalLinks(err error) []household.ShareLink {
	var linked *household.LinkedPersonalError
	if errors.As(err, &linked) {
		return linked.Links
	}
	return nil
}
