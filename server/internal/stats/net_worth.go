package stats

import (
	"context"
	"sort"
	"time"

	"github.com/kai-zer-ru/buhgalter/internal/account"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type NetWorthSnapshot struct {
	Period                  string `json:"period,omitempty"`
	AsOf                    string `json:"as_of"`
	NetWorth                int64  `json:"net_worth"`
	NetWorthDisplay         string `json:"net_worth_display"`
	Assets                  int64  `json:"assets"`
	AssetsDisplay           string `json:"assets_display"`
	Liabilities             int64  `json:"liabilities"`
	LiabilitiesDisplay      string `json:"liabilities_display"`
	CashBank                int64  `json:"cash_bank"`
	CashBankDisplay         string `json:"cash_bank_display"`
	OwedToMe                int64  `json:"owed_to_me"`
	OwedToMeDisplay         string `json:"owed_to_me_display"`
	IOwe                    int64  `json:"i_owe"`
	IOweDisplay             string `json:"i_owe_display"`
	CreditsRemaining        int64  `json:"credits_remaining"`
	CreditsRemainingDisplay string `json:"credits_remaining_display"`
	CreditCardsDebt         int64  `json:"credit_cards_debt"`
	CreditCardsDebtDisplay  string `json:"credit_cards_debt_display"`
}

type NetWorthResponse struct {
	Current NetWorthSnapshot   `json:"current"`
	Items   []NetWorthSnapshot `json:"items"`
}

const maxNetWorthMonths = 120

func (s *Service) NetWorth(ctx context.Context, userID, from, to, groupBy string) (NetWorthResponse, error) {
	if groupBy == "" {
		groupBy = "month"
	}
	if groupBy != "month" {
		return NetWorthResponse{}, ErrInvalidGroupBy
	}
	tz, err := queries(s.db).GetUserTimezone(ctx, userID)
	if err != nil {
		return NetWorthResponse{}, err
	}
	if tz == "" {
		tz = "Europe/Moscow"
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	now := timeutil.NowUTC()
	rangeFrom, rangeTo, err := netWorthMonthRange(from, to, now, loc)
	if err != nil {
		return NetWorthResponse{}, err
	}
	if monthCount(rangeFrom, rangeTo) > maxNetWorthMonths {
		return NetWorthResponse{}, ErrInvalidDate
	}

	current, err := s.snapshotAt(ctx, userID, timeutil.FormatUTC(now), "")
	if err != nil {
		return NetWorthResponse{}, err
	}

	items := make([]NetWorthSnapshot, 0)
	for monthStart := rangeFrom; !monthStart.After(rangeTo); monthStart = monthStart.AddDate(0, 1, 0) {
		periodKey := monthStart.Format("2006-01-02")
		asOf := monthEndInTZ(monthStart, loc, now)
		snap, err := s.snapshotAt(ctx, userID, asOf, periodKey)
		if err != nil {
			return NetWorthResponse{}, err
		}
		items = append(items, snap)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Period > items[j].Period
	})
	return NetWorthResponse{Current: current, Items: items}, nil
}

func monthEndInTZ(monthStart time.Time, loc *time.Location, now time.Time) string {
	year, month, _ := monthStart.Date()
	end := time.Date(year, month+1, 0, 23, 59, 59, 0, loc)
	endUTC := end.UTC()
	if endUTC.After(now) {
		return timeutil.FormatUTC(now)
	}
	return timeutil.FormatUTC(endUTC)
}

func netWorthMonthRange(from, to string, now time.Time, loc *time.Location) (time.Time, time.Time, error) {
	inTZ := now.In(loc)
	year, month, _ := inTZ.Date()
	defaultEnd := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	defaultStart := defaultEnd.AddDate(0, -11, 0)

	var startMonth, endMonth time.Time
	if from != "" {
		t, err := timeutil.ParseUTC(from)
		if err != nil {
			return time.Time{}, time.Time{}, ErrInvalidDate
		}
		local := t.In(loc)
		startMonth = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	} else {
		startMonth = defaultStart
	}
	if to != "" {
		t, err := timeutil.ParseUTC(to)
		if err != nil {
			return time.Time{}, time.Time{}, ErrInvalidDate
		}
		local := t.In(loc)
		endMonth = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	} else {
		endMonth = defaultEnd
	}
	if startMonth.After(endMonth) {
		return time.Time{}, time.Time{}, ErrInvalidDate
	}
	return startMonth, endMonth, nil
}

func monthCount(rangeFrom, rangeTo time.Time) int {
	n := 0
	for monthStart := rangeFrom; !monthStart.After(rangeTo); monthStart = monthStart.AddDate(0, 1, 0) {
		n++
	}
	return n
}

func (s *Service) snapshotAt(ctx context.Context, userID, asOf, period string) (NetWorthSnapshot, error) {
	q := queries(s.db)
	accRows, err := q.ListAccountsNetWorthByUser(ctx, userID)
	if err != nil {
		return NetWorthSnapshot{}, err
	}
	asOfTime, err := timeutil.ParseUTC(asOf)
	if err != nil {
		return NetWorthSnapshot{}, ErrInvalidDate
	}
	balances, err := accountbalance.ComputeAllAsOf(ctx, s.db, userID, asOf)
	if err != nil {
		return NetWorthSnapshot{}, err
	}

	var cashBank, creditCardsDebt int64
	for _, row := range accRows {
		created, err := timeutil.ParseFlexibleUTC(row.CreatedAt)
		if err != nil {
			return NetWorthSnapshot{}, err
		}
		if created.After(asOfTime) {
			continue
		}
		bal := balances[row.ID]
		switch {
		case account.IsCreditCard(row.Type):
			if row.CreditLimit != nil {
				used := *row.CreditLimit - bal
				if used > 0 {
					creditCardsDebt += used
				}
			}
		case row.Type == "cash" || row.Type == "bank":
			cashBank += bal
		}
	}

	debtRows, err := q.NetWorthDebtsByUser(ctx, sqlcdb.NetWorthDebtsByUserParams{
		UserID: userID,
		AsOf:   asOf,
	})
	if err != nil {
		return NetWorthSnapshot{}, err
	}
	var owedToMe, iOwe int64
	for _, row := range debtRows {
		if row.AmountAtAsOf <= 0 {
			continue
		}
		switch row.Direction {
		case "lent":
			owedToMe += row.AmountAtAsOf
		case "borrowed":
			iOwe += row.AmountAtAsOf
		}
	}

	creditRows, err := q.NetWorthCreditsByUser(ctx, sqlcdb.NetWorthCreditsByUserParams{
		UserID: userID,
		AsOf:   asOf,
	})
	if err != nil {
		return NetWorthSnapshot{}, err
	}
	var creditsRemaining int64
	for _, row := range creditRows {
		rem := row.PrincipalAmount - row.PaidAtAsOf
		if rem > 0 {
			creditsRemaining += rem
		}
	}

	assets := cashBank + owedToMe
	liabilities := iOwe + creditsRemaining + creditCardsDebt
	netWorth := assets - liabilities

	return NetWorthSnapshot{
		Period:                  period,
		AsOf:                    asOf,
		NetWorth:                netWorth,
		NetWorthDisplay:         money.FormatRubles(netWorth),
		Assets:                  assets,
		AssetsDisplay:           money.FormatRubles(assets),
		Liabilities:             liabilities,
		LiabilitiesDisplay:      money.FormatRubles(liabilities),
		CashBank:                cashBank,
		CashBankDisplay:         money.FormatRubles(cashBank),
		OwedToMe:                owedToMe,
		OwedToMeDisplay:         money.FormatRubles(owedToMe),
		IOwe:                    iOwe,
		IOweDisplay:             money.FormatRubles(iOwe),
		CreditsRemaining:        creditsRemaining,
		CreditsRemainingDisplay: money.FormatRubles(creditsRemaining),
		CreditCardsDebt:         creditCardsDebt,
		CreditCardsDebtDisplay:  money.FormatRubles(creditCardsDebt),
	}, nil
}
