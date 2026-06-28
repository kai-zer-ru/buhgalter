package credit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kai-zer-ru/buhgalter/internal/accountbalance"
	"github.com/kai-zer-ru/buhgalter/internal/categoryseed"
	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
	"github.com/kai-zer-ru/buhgalter/internal/money"
	"github.com/kai-zer-ru/buhgalter/internal/timeutil"
)

type Credit struct {
	ID                        string          `json:"id"`
	Name                      *string         `json:"name"`
	CreditKind                string          `json:"credit_kind"`
	PrincipalAmount           int64           `json:"principal_amount"`
	PrincipalAmountDisplay    string          `json:"principal_amount_display"`
	PropertyPrice             *int64          `json:"property_price,omitempty"`
	PropertyPriceDisplay      *string         `json:"property_price_display,omitempty"`
	DownPayment               int64           `json:"down_payment"`
	DownPaymentDisplay        string          `json:"down_payment_display"`
	DownPaymentAffectsBalance bool            `json:"down_payment_affects_balance"`
	DownPaymentTransactionID  *string         `json:"down_payment_transaction_id,omitempty"`
	IssueDate                 string          `json:"issue_date"`
	TermMonths                int             `json:"term_months"`
	InterestRate              float64         `json:"interest_rate"`
	PaymentInterval           string          `json:"payment_interval"`
	PaidAmount                int64           `json:"paid_amount"`
	PaidAmountDisplay         string          `json:"paid_amount_display"`
	MonthlyPayment            int64           `json:"monthly_payment"`
	MonthlyPaymentDisplay     string          `json:"monthly_payment_display"`
	CalculatedMonthlyPayment  *int64          `json:"calculated_monthly_payment,omitempty"`
	RemainingAmount           int64           `json:"remaining_amount"`
	RemainingAmountDisplay    string          `json:"remaining_amount_display"`
	DebitAccountID            string          `json:"debit_account_id"`
	DebitAccountName          string          `json:"debit_account_name"`
	DebitTimeLocal            *string         `json:"debit_time_local,omitempty"`
	BankID                    *string         `json:"bank_id,omitempty"`
	BankName                  *string         `json:"bank_name,omitempty"`
	BankIDLocked              bool            `json:"bank_id_locked"`
	AddedRetroactively        bool            `json:"added_retroactively"`
	RecordedAt                string          `json:"recorded_at"`
	Status                    string          `json:"status"`
	ClosedAt                  *string         `json:"closed_at"`
	IsInstallment             bool            `json:"is_installment"`
	NextPaymentDate           *string         `json:"next_payment_date,omitempty"`
	NextPaymentAmount         *int64          `json:"next_payment_amount,omitempty"`
	Schedule                  []CreditPayment `json:"schedule,omitempty"`
	SchedulePreview           []ScheduleEntry `json:"schedule_preview,omitempty"`
	CreatedAt                 string          `json:"created_at"`
	UpdatedAt                 string          `json:"updated_at"`
}

type CreditPayment struct {
	ID               string  `json:"id"`
	CreditID         string  `json:"credit_id"`
	TransactionID    *string `json:"transaction_id"`
	TransactionKind  *string `json:"transaction_kind,omitempty"`
	Amount           int64   `json:"amount"`
	AmountDisplay    string  `json:"amount_display"`
	PaymentDate      string  `json:"payment_date"`
	Kind             string  `json:"kind"`
	IsApplied        bool    `json:"is_applied"`
	ExcludeFromStats bool    `json:"exclude_from_stats"`
	CreatedAt        string  `json:"created_at"`
}

type CreateInput struct {
	Name                      *string
	CreditKind                string
	PrincipalAmount           int64
	PropertyPrice             *int64
	DownPayment               int64
	DownPaymentAffectsBalance bool
	DownPaymentAccountID      *string
	IssueDate                 time.Time
	TermMonths                int
	InterestRate              float64
	PaymentInterval           PaymentInterval
	PaidAmount                int64
	MonthlyPayment            *int64
	DebitAccountID            string
	DebitTimeLocal            *string
	BankID                    *string
	AddedRetroactively        *bool
	RetroactiveDebitCount     int
	ScheduleSeed              []ScheduleSeed
	CreateTransactions        bool
}

type UpdateInput struct {
	Name           *string
	MonthlyPayment *int64
	DebitAccountID *string
	DebitTimeLocal *string
	BankID         *string
	BankIDSet      bool
}

type PayPaymentInput struct {
	Amount      int64
	PaymentDate time.Time
	AccountID   string
}

type CompleteInput struct {
	AffectsBalance bool
	PaymentDate    time.Time
}

var (
	ErrNotFound                = errors.New("credit not found")
	ErrInvalidAmount           = errors.New("invalid amount")
	ErrInvalidTerm             = errors.New("invalid term")
	ErrInvalidAccount          = errors.New("invalid account")
	ErrAccountArchived         = errors.New("account is archived")
	ErrAlreadyClosed           = errors.New("credit already closed")
	ErrInvalidInterval         = errors.New("invalid payment interval")
	ErrInvalidPaymentDate      = errors.New("invalid payment date")
	ErrPaymentApplied          = errors.New("payment already applied")
	ErrNoPendingPayment        = errors.New("no pending scheduled payment")
	ErrCannotRemoveRetroactive = errors.New("cannot remove retroactive payment")
	ErrOnlyLatestPaymentDelete = errors.New("only latest applied payment can be deleted")
	ErrInvalidRetroactiveDebit = errors.New("invalid retroactive debit count")
	ErrCompleteParamsRequired  = errors.New("complete parameters required")
	ErrInvalidDebitTime        = errors.New("invalid debit time")
	ErrCreditBankLocked        = errors.New("credit bank can be edited only once")
	ErrInvalidBank             = errors.New("invalid bank")
	ErrPlannedNotAllowed       = errors.New("planned operation is not allowed for credit payments")
	ErrInvalidCreditKind       = errors.New("invalid credit kind")
	ErrInvalidMortgageFields   = errors.New("invalid mortgage fields")
)

const (
	CreditKindConsumer = "consumer"
	CreditKindMortgage = "mortgage"
)

var localTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

type PreviewInput struct {
	Principal       int64
	TermMonths      int
	InterestRate    float64
	PaymentInterval PaymentInterval
	CreditKind      string
	IssueDate       time.Time
	MonthlyPayment  *int64
	SeedPayments    []ScheduleSeed
}

func queries(db sqlcdb.DBTX) *sqlcdb.Queries {
	return sqlcdb.New(db)
}

func List(ctx context.Context, db *sql.DB, userID, statusFilter string) ([]Credit, error) {
	rows, err := queries(db).ListCreditsByUser(ctx, sqlcdb.ListCreditsByUserParams{
		UserID: userID, Column2: statusFilter, Status: statusFilter,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Credit, 0, len(rows))
	for _, row := range rows {
		c, err := creditFromListRow(ctx, db, userID, row)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

func GetByID(ctx context.Context, db *sql.DB, userID, id string, withSchedule bool) (Credit, error) {
	row, err := queries(db).GetCreditByID(ctx, sqlcdb.GetCreditByIDParams{ID: id, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return Credit{}, ErrNotFound
	}
	if err != nil {
		return Credit{}, err
	}
	c, err := creditFromGetRow(ctx, db, userID, row)
	if err != nil {
		return Credit{}, err
	}
	if withSchedule {
		schedule, err := loadSchedule(ctx, db, id)
		if err != nil {
			return Credit{}, err
		}
		c.Schedule = schedule
	}
	return c, nil
}

// monthlyPaymentMatchTolerance allows rounding to whole rubles without treating payment as custom.
const monthlyPaymentMatchTolerance = int64(100)

func calculatedMonthlyPayment(principal int64, termMonths int, interestRate float64, creditKind string, interval PaymentInterval, issueDate time.Time) int64 {
	monthly := MonthlyPayment(principal, interestRate, termMonths)
	if normalizeCreditKind(creditKind) == CreditKindMortgage && interval == IntervalMonth {
		monthly = MonthlyPaymentMortgage(principal, interestRate, termMonths, issueDate)
	}
	return monthly
}

func resolveScheduleMonthly(principal int64, termMonths int, interestRate float64, creditKind string, interval PaymentInterval, issueDate time.Time, userMonthly *int64) (monthly int64, userSet bool) {
	monthly = calculatedMonthlyPayment(principal, termMonths, interestRate, creditKind, interval, issueDate)
	if userMonthly == nil {
		return monthly, false
	}
	diff := *userMonthly - monthly
	if diff < 0 {
		diff = -diff
	}
	if diff <= monthlyPaymentMatchTolerance {
		return monthly, false
	}
	return *userMonthly, true
}

func validateUserMonthlyAboveCalculated(calculated, monthly int64) error {
	if calculated <= 0 || monthly <= calculated {
		return nil
	}
	// Allow up to 50% above auto-calculated payment.
	if monthly > calculated+calculated/2 {
		return ErrMonthlyPaymentTooHighForTerm
	}
	return nil
}

func PreviewSchedule(in PreviewInput) ([]ScheduleEntry, int64, error) {
	monthly, userSet := resolveScheduleMonthly(
		in.Principal, in.TermMonths, in.InterestRate, in.CreditKind, in.PaymentInterval, in.IssueDate, in.MonthlyPayment,
	)
	if userSet && normalizeCreditKind(in.CreditKind) != CreditKindMortgage {
		calculated := calculatedMonthlyPayment(in.Principal, in.TermMonths, in.InterestRate, in.CreditKind, in.PaymentInterval, in.IssueDate)
		if err := validateUserMonthlyAboveCalculated(calculated, monthly); err != nil {
			return nil, 0, err
		}
	}
	entries, err := GenerateSchedule(ScheduleInput{
		Principal:       in.Principal,
		TermMonths:      in.TermMonths,
		MonthlyPayment:  monthly,
		UserSetPayment:  userSet,
		PaymentInterval: in.PaymentInterval,
		CreditKind:      normalizeCreditKind(in.CreditKind),
		IssueDate:       in.IssueDate,
		InterestRate:    in.InterestRate,
		SeedPayments:    in.SeedPayments,
	})
	if err != nil {
		return nil, 0, err
	}
	for i := range entries {
		entries[i].AmountDisplay = money.FormatRubles(entries[i].Amount)
	}
	return entries, monthly, nil
}

func Create(ctx context.Context, db *sql.DB, userID string, in CreateInput) (Credit, error) {
	creditKind := normalizeCreditKind(in.CreditKind)
	if !isValidCreditKind(creditKind) {
		return Credit{}, ErrInvalidCreditKind
	}
	principalAmount, propertyPrice, downPayment, err := normalizeCreditAmounts(creditKind, in.PrincipalAmount, in.PropertyPrice, in.DownPayment)
	if err != nil {
		return Credit{}, err
	}
	if principalAmount <= 0 {
		return Credit{}, ErrInvalidAmount
	}
	if in.TermMonths <= 0 {
		return Credit{}, ErrInvalidTerm
	}
	if !in.PaymentInterval.Valid() {
		return Credit{}, ErrInvalidInterval
	}
	if err := validateActiveAccount(ctx, db, userID, in.DebitAccountID); err != nil {
		return Credit{}, err
	}
	debitTimeLocal, err := normalizeDebitTimeLocal(in.DebitTimeLocal)
	if err != nil {
		return Credit{}, err
	}
	bankID := normalizeNullableID(in.BankID)
	if err := validateBank(ctx, db, bankID); err != nil {
		return Credit{}, err
	}

	if in.MonthlyPayment != nil && *in.MonthlyPayment <= 0 {
		return Credit{}, ErrInvalidAmount
	}
	monthly, userSet := resolveScheduleMonthly(
		principalAmount, in.TermMonths, in.InterestRate, creditKind, in.PaymentInterval, in.IssueDate, in.MonthlyPayment,
	)
	if userSet && creditKind != CreditKindMortgage {
		calculated := calculatedMonthlyPayment(principalAmount, in.TermMonths, in.InterestRate, creditKind, in.PaymentInterval, in.IssueDate)
		if err := validateUserMonthlyAboveCalculated(calculated, monthly); err != nil {
			return Credit{}, err
		}
	}
	if in.PaymentInterval == IntervalManual && len(in.ScheduleSeed) > 0 {
		var sum int64
		for _, s := range in.ScheduleSeed {
			sum += s.Amount
		}
		monthly = sum / int64(len(in.ScheduleSeed))
	}

	entries, err := GenerateSchedule(ScheduleInput{
		Principal:       principalAmount,
		TermMonths:      in.TermMonths,
		MonthlyPayment:  monthly,
		UserSetPayment:  userSet,
		PaymentInterval: in.PaymentInterval,
		IssueDate:       in.IssueDate,
		InterestRate:    in.InterestRate,
		CreditKind:      creditKind,
		SeedPayments:    in.ScheduleSeed,
	})
	if err != nil {
		return Credit{}, err
	}

	tz, err := userTimezone(ctx, db, userID)
	if err != nil {
		return Credit{}, err
	}
	todayStart, err := timeutil.TodayStartUTC(tz, timeutil.NowUTC())
	if err != nil {
		return Credit{}, err
	}

	addedRetro := false
	if in.AddedRetroactively != nil {
		addedRetro = *in.AddedRetroactively
	} else if in.IssueDate.Before(todayStart) {
		addedRetro = true
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	recordedAt := timeutil.FormatUTC(now)
	issueDate := timeutil.FormatUTC(in.IssueDate)

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Credit{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)

	paidAmount := in.PaidAmount
	addedRetroInt := int64(0)
	if addedRetro {
		addedRetroInt = 1
	}

	if err := q.InsertCredit(ctx, sqlcdb.InsertCreditParams{
		ID: id, UserID: userID, Name: in.Name,
		CreditKind:      creditKind,
		PrincipalAmount: principalAmount, PropertyPrice: propertyPrice, DownPayment: downPayment,
		DownPaymentAffectsBalance: boolToInt64(in.DownPaymentAffectsBalance),
		DownPaymentTransactionID:  nil,
		IssueDate:                 issueDate,
		TermMonths:                int64(in.TermMonths), InterestRate: in.InterestRate,
		PaymentInterval: string(in.PaymentInterval), PaidAmount: paidAmount,
		MonthlyPayment: monthly, DebitAccountID: in.DebitAccountID,
		DebitTimeLocal: debitTimeLocal, BankID: bankID, BankIDLocked: 0,
		AddedRetroactively: addedRetroInt, RecordedAt: recordedAt,
		Status: "active", ClosedAt: nil, CreatedAt: nowStr, UpdatedAt: nowStr,
	}); err != nil {
		return Credit{}, err
	}

	debitRetro, err := retroactiveDebitEntrySet(entries, addedRetro, todayStart, in.RetroactiveDebitCount)
	if err != nil {
		return Credit{}, err
	}

	var retroPaid int64
	var downPaymentTxID *string
	if creditKind == CreditKindMortgage && downPayment > 0 && in.DownPaymentAffectsBalance {
		downPaymentAccountID := in.DebitAccountID
		if in.DownPaymentAccountID != nil && strings.TrimSpace(*in.DownPaymentAccountID) != "" {
			if err := validateActiveAccount(ctx, db, userID, *in.DownPaymentAccountID); err != nil {
				return Credit{}, err
			}
			downPaymentAccountID = *in.DownPaymentAccountID
		}
		tid, err := insertCreditExpenseTransaction(ctx, dbTx, userID, downPaymentAccountID, in.Name, downPayment, in.IssueDate, true)
		if err != nil {
			return Credit{}, err
		}
		downPaymentTxID = &tid
	}
	for i, e := range entries {
		payDate, err := timeutil.ParseUTC(e.PaymentDate)
		if err != nil {
			return Credit{}, err
		}
		kind := "scheduled"
		applied := int64(0)
		excludeStats := int64(0)
		if addedRetro && payDate.Before(todayStart) {
			kind = "retroactive"
			applied = 1
			excludeStats = 1
			retroPaid += e.Amount
		}

		var txID *string
		if _, debitRetro := debitRetro[i]; debitRetro {
			tid, err := insertCreditExpenseTransaction(ctx, dbTx, userID, in.DebitAccountID, in.Name, e.Amount, payDate, true)
			if err != nil {
				return Credit{}, err
			}
			txID = &tid
			excludeStats = 0
		}
		// Auto-debit does not precreate future transactions.
		// Transactions are created only when the scheduled moment is reached by the scheduler.

		if err := q.InsertCreditPayment(ctx, sqlcdb.InsertCreditPaymentParams{
			ID: uuid.NewString(), CreditID: id, TransactionID: txID,
			Amount: e.Amount, PaymentDate: e.PaymentDate, Kind: kind,
			IsApplied: applied, ExcludeFromStats: excludeStats, CreatedAt: nowStr,
		}); err != nil {
			return Credit{}, err
		}
	}

	paidAmount = in.PaidAmount + retroPaid
	if paidAmount > principalAmount {
		paidAmount = principalAmount
	}
	if paidAmount != in.PaidAmount {
		if err := q.UpdateCreditPaidAmount(ctx, sqlcdb.UpdateCreditPaidAmountParams{
			PaidAmount: paidAmount, UpdatedAt: nowStr, ID: id, UserID: userID,
		}); err != nil {
			return Credit{}, err
		}
	}
	if downPaymentTxID != nil {
		if err := q.SetCreditDownPaymentTransaction(ctx, sqlcdb.SetCreditDownPaymentTransactionParams{
			DownPaymentTransactionID: downPaymentTxID,
			ID:                       id,
			UserID:                   userID,
		}); err != nil {
			return Credit{}, err
		}
	}

	if err := dbTx.Commit(); err != nil {
		return Credit{}, err
	}
	syncAccountBalances(ctx, db, userID, in.DebitAccountID)
	if creditKind == CreditKindMortgage && downPayment > 0 && in.DownPaymentAffectsBalance {
		downPaymentAccountID := in.DebitAccountID
		if in.DownPaymentAccountID != nil && strings.TrimSpace(*in.DownPaymentAccountID) != "" {
			downPaymentAccountID = *in.DownPaymentAccountID
		}
		if downPaymentAccountID != in.DebitAccountID {
			syncAccountBalances(ctx, db, userID, downPaymentAccountID)
		}
	}
	return GetByID(ctx, db, userID, id, true)
}

func retroactiveDebitEntrySet(entries []ScheduleEntry, addedRetro bool, todayStart time.Time, count int) (map[int]struct{}, error) {
	out := make(map[int]struct{})
	if count == 0 {
		return out, nil
	}
	if !addedRetro || count < 0 {
		return nil, ErrInvalidRetroactiveDebit
	}
	var retroIdx []int
	for i, e := range entries {
		payDate, err := timeutil.ParseUTC(e.PaymentDate)
		if err != nil {
			return nil, err
		}
		if payDate.Before(todayStart) {
			retroIdx = append(retroIdx, i)
		}
	}
	if count > len(retroIdx) {
		return nil, ErrInvalidRetroactiveDebit
	}
	for _, i := range retroIdx[len(retroIdx)-count:] {
		out[i] = struct{}{}
	}
	return out, nil
}

func Update(ctx context.Context, db *sql.DB, userID, id string, in UpdateInput) (Credit, error) {
	existing, err := GetByID(ctx, db, userID, id, false)
	if err != nil {
		return Credit{}, err
	}
	if existing.Status == "closed" {
		return Credit{}, ErrAlreadyClosed
	}

	name := existing.Name
	if in.Name != nil {
		name = in.Name
	}
	monthly := existing.MonthlyPayment
	if in.MonthlyPayment != nil {
		if *in.MonthlyPayment <= 0 {
			return Credit{}, ErrInvalidAmount
		}
		monthly = *in.MonthlyPayment
	}
	debitAccount := existing.DebitAccountID
	if in.DebitAccountID != nil {
		if err := validateActiveAccount(ctx, db, userID, *in.DebitAccountID); err != nil {
			return Credit{}, err
		}
		debitAccount = *in.DebitAccountID
	}
	debitTimeLocal := existing.DebitTimeLocal
	if in.DebitTimeLocal != nil {
		normalized, err := normalizeDebitTimeLocal(in.DebitTimeLocal)
		if err != nil {
			return Credit{}, err
		}
		debitTimeLocal = normalized
	}
	bankID := normalizeNullableID(existing.BankID)
	if in.BankIDSet {
		nextBankID := normalizeNullableID(in.BankID)
		if err := validateBank(ctx, db, nextBankID); err != nil {
			return Credit{}, err
		}
		bankID = nextBankID
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Credit{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	nowStr := time.Now().UTC().Format(time.RFC3339)
	n, err := q.UpdateCredit(ctx, sqlcdb.UpdateCreditParams{
		Name:           name,
		MonthlyPayment: monthly,
		DebitAccountID: debitAccount,
		DebitTimeLocal: debitTimeLocal,
		BankID:         bankID,
		BankIDLocked:   0,
		UpdatedAt:      nowStr,
		ID:             id,
		UserID:         userID,
	})
	if err != nil {
		return Credit{}, err
	}
	if n == 0 {
		return Credit{}, ErrNotFound
	}

	if in.DebitAccountID != nil && *in.DebitAccountID != existing.DebitAccountID {
		if err := updateFutureTransactionAccounts(ctx, dbTx, userID, id, *in.DebitAccountID, nowStr); err != nil {
			return Credit{}, err
		}
	}

	if err := dbTx.Commit(); err != nil {
		return Credit{}, err
	}
	return GetByID(ctx, db, userID, id, true)
}

func PayNextScheduled(ctx context.Context, db *sql.DB, userID, creditID string, in PayPaymentInput) (Credit, error) {
	if in.Amount <= 0 {
		return Credit{}, ErrInvalidAmount
	}
	c, err := GetByID(ctx, db, userID, creditID, true)
	if err != nil {
		return Credit{}, err
	}
	if c.Status == "closed" {
		return Credit{}, ErrAlreadyClosed
	}
	remaining := RemainingAmount(c.PrincipalAmount, c.PaidAmount)
	if in.Amount > remaining {
		return Credit{}, ErrInvalidAmount
	}
	debitAccountID := c.DebitAccountID
	if in.AccountID != "" {
		debitAccountID = in.AccountID
	}
	if err := validateActiveAccount(ctx, db, userID, debitAccountID); err != nil {
		return Credit{}, err
	}

	var targetID string
	var existingTxID *string
	for _, p := range c.Schedule {
		if p.Kind == "scheduled" && !p.IsApplied {
			targetID = p.ID
			existingTxID = p.TransactionID
			break
		}
	}
	if targetID == "" {
		return Credit{}, ErrNoPendingPayment
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Credit{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	nowStr := time.Now().UTC().Format(time.RFC3339)
	payDate := timeutil.FormatUTC(in.PaymentDate)
	txKind, err := resolveTransactionKind(ctx, dbTx, userID, in.PaymentDate)
	if err != nil {
		return Credit{}, err
	}
	catID, err := categoryseed.CreditCategoryID(ctx, dbTx, userID)
	if err != nil {
		return Credit{}, err
	}
	desc := creditDescription(c.Name)
	descPtr := &desc

	var txID string
	if existingTxID != nil {
		txID = *existingTxID
		if err := q.UpdateTransaction(ctx, sqlcdb.UpdateTransactionParams{
			AccountID:       debitAccountID,
			Type:            "expense",
			Kind:            txKind,
			Amount:          in.Amount,
			Description:     descPtr,
			CategoryID:      &catID,
			SubcategoryID:   nil,
			TransactionDate: payDate,
			UpdatedAt:       nowStr,
			ID:              txID,
			UserID:          userID,
		}); err != nil {
			return Credit{}, err
		}
		if _, err := q.UpdateTransactionAffectsBalance(ctx, sqlcdb.UpdateTransactionAffectsBalanceParams{
			AffectsBalance: 1,
			UpdatedAt:      nowStr,
			ID:             txID,
			UserID:         userID,
		}); err != nil {
			return Credit{}, err
		}
	} else {
		txID, err = insertCreditExpenseTransaction(ctx, dbTx, userID, debitAccountID, c.Name, in.Amount, in.PaymentDate, true)
		if err != nil {
			return Credit{}, err
		}
	}

	n, err := q.ApplyScheduledPayment(ctx, sqlcdb.ApplyScheduledPaymentParams{
		TransactionID: &txID,
		PaymentDate:   payDate,
		Amount:        in.Amount,
		ID:            targetID,
		CreditID:      creditID,
	})
	if err != nil {
		return Credit{}, err
	}
	if n == 0 {
		return Credit{}, ErrNoPendingPayment
	}

	newPaid := c.PaidAmount + in.Amount
	if err := q.UpdateCreditPaidAmount(ctx, sqlcdb.UpdateCreditPaidAmountParams{
		PaidAmount: newPaid, UpdatedAt: nowStr, ID: creditID, UserID: userID,
	}); err != nil {
		return Credit{}, err
	}

	if err := maybeAutoClose(ctx, dbTx, userID, creditID, c.PrincipalAmount, newPaid, nowStr); err != nil {
		return Credit{}, err
	}

	if err := cleanupPaymentDayDuplicates(ctx, dbTx, creditID, in.PaymentDate, userID, targetID); err != nil {
		return Credit{}, err
	}

	if err := dbTx.Commit(); err != nil {
		return Credit{}, err
	}
	syncAccountBalances(ctx, db, userID, debitAccountID)
	return GetByID(ctx, db, userID, creditID, true)
}

func Complete(ctx context.Context, db *sql.DB, userID, id string, in CompleteInput) (Credit, error) {
	c, err := GetByID(ctx, db, userID, id, true)
	if err != nil {
		return Credit{}, err
	}
	if c.Status == "closed" {
		return Credit{}, ErrAlreadyClosed
	}

	remaining := RemainingAmount(c.PrincipalAmount, c.PaidAmount)
	if remaining > 0 && in.PaymentDate.IsZero() {
		return Credit{}, ErrCompleteParamsRequired
	}

	paymentDate := in.PaymentDate
	if paymentDate.IsZero() {
		paymentDate = timeutil.NowUTC()
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Credit{}, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	if err := cancelUnappliedPayments(ctx, dbTx, userID, id); err != nil {
		return Credit{}, err
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	payDateStr := timeutil.FormatUTC(paymentDate)

	if remaining > 0 {
		if in.AffectsBalance {
			if err := validateActiveAccount(ctx, db, userID, c.DebitAccountID); err != nil {
				return Credit{}, err
			}
			txID, err := insertCreditExpenseTransaction(ctx, dbTx, userID, c.DebitAccountID, c.Name, remaining, paymentDate, true)
			if err != nil {
				return Credit{}, err
			}
			if err := q.InsertCreditPayment(ctx, sqlcdb.InsertCreditPaymentParams{
				ID: uuid.NewString(), CreditID: id, TransactionID: &txID,
				Amount: remaining, PaymentDate: payDateStr, Kind: "auto",
				IsApplied: 1, ExcludeFromStats: 0, CreatedAt: nowStr,
			}); err != nil {
				return Credit{}, err
			}
		} else {
			txID, err := insertCreditExpenseTransaction(ctx, dbTx, userID, c.DebitAccountID, c.Name, remaining, paymentDate, false)
			if err != nil {
				return Credit{}, err
			}
			if err := q.InsertCreditPayment(ctx, sqlcdb.InsertCreditPaymentParams{
				ID: uuid.NewString(), CreditID: id, TransactionID: &txID,
				Amount: remaining, PaymentDate: payDateStr, Kind: "auto",
				IsApplied: 1, ExcludeFromStats: 0, CreatedAt: nowStr,
			}); err != nil {
				return Credit{}, err
			}
		}
	}

	if c.PaidAmount != c.PrincipalAmount {
		if err := q.UpdateCreditPaidAmount(ctx, sqlcdb.UpdateCreditPaidAmountParams{
			PaidAmount: c.PrincipalAmount, UpdatedAt: nowStr, ID: id, UserID: userID,
		}); err != nil {
			return Credit{}, err
		}
	}

	closedAt := payDateStr
	n, err := q.CloseCredit(ctx, sqlcdb.CloseCreditParams{
		ClosedAt: &closedAt, UpdatedAt: nowStr, ID: id, UserID: userID,
	})
	if err != nil {
		return Credit{}, err
	}
	if n == 0 {
		return Credit{}, ErrNotFound
	}

	if err := dbTx.Commit(); err != nil {
		return Credit{}, err
	}
	if remaining > 0 && in.AffectsBalance {
		syncAccountBalances(ctx, db, userID, c.DebitAccountID)
	}
	return GetByID(ctx, db, userID, id, true)
}

// Close completes a credit with no remaining balance (used by auto-close).
func Close(ctx context.Context, db *sql.DB, userID, id string) (Credit, error) {
	return Complete(ctx, db, userID, id, CompleteInput{
		AffectsBalance: false,
		PaymentDate:    timeutil.NowUTC(),
	})
}

func Delete(ctx context.Context, db *sql.DB, userID, id, mode string) error {
	if mode != "cascade" && mode != "keep_transactions" {
		mode = "cascade"
	}
	_, err := GetByID(ctx, db, userID, id, false)
	if err != nil {
		return err
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	creditRow, err := q.GetCreditByID(ctx, sqlcdb.GetCreditByIDParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	txIDs, err := q.ListCreditPaymentTransactionIDs(ctx, id)
	if err != nil {
		return err
	}
	if creditRow.DownPaymentTransactionID != nil {
		txIDs = append(txIDs, creditRow.DownPaymentTransactionID)
	}

	if mode == "keep_transactions" {
		suffix := " (кредит удалён)"
		nowStr := time.Now().UTC().Format(time.RFC3339)
		for _, txID := range txIDs {
			if txID == nil {
				continue
			}
			row, err := q.GetTransactionByID(ctx, sqlcdb.GetTransactionByIDParams{ID: *txID, UserID: userID})
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return err
			}
			desc := ""
			if row.Description != nil {
				desc = *row.Description
			}
			if !strings.Contains(desc, suffix) {
				desc += suffix
			}
			if err := q.UpdateTransaction(ctx, sqlcdb.UpdateTransactionParams{
				AccountID: row.AccountID, Type: row.Type, Kind: row.Kind,
				Amount: row.Amount, Description: &desc,
				CategoryID: row.CategoryID, SubcategoryID: row.SubcategoryID,
				TransactionDate: row.TransactionDate, UpdatedAt: nowStr,
				ID: *txID, UserID: userID,
			}); err != nil {
				return err
			}
		}
		if err := q.UnlinkCreditPaymentTransactions(ctx, id); err != nil {
			return err
		}
	} else {
		if err := q.UnlinkCreditPaymentTransactions(ctx, id); err != nil {
			return err
		}
		for _, txID := range txIDs {
			if txID == nil {
				continue
			}
			if _, err := q.DeleteTransaction(ctx, sqlcdb.DeleteTransactionParams{ID: *txID, UserID: userID}); err != nil {
				return err
			}
		}
	}

	n, err := q.DeleteCredit(ctx, sqlcdb.DeleteCreditParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	if err := dbTx.Commit(); err != nil {
		return err
	}
	if mode == "cascade" {
		syncAccountBalances(ctx, db, userID, creditRow.DebitAccountID)
	}
	return nil
}

func ApplyDuePayments(ctx context.Context, db *sql.DB, userID string, todayCutoff string, localTime string) (int, error) {
	rows, err := queries(db).ListDueCreditPayments(ctx, todayCutoff)
	if err != nil {
		return 0, err
	}
	applied := 0
	for _, row := range rows {
		if row.UserID != userID {
			continue
		}
		if row.DebitTimeLocal == nil {
			continue
		}
		if strings.TrimSpace(*row.DebitTimeLocal) != localTime {
			continue
		}
		ok, err := processAutoPayment(ctx, db, row)
		if err != nil {
			return applied, err
		}
		if ok {
			applied++
		}
	}
	return applied, nil
}

func processAutoPayment(ctx context.Context, db *sql.DB, row sqlcdb.ListDueCreditPaymentsRow) (bool, error) {
	count, err := queries(db).HasPaymentOnDate(ctx, sqlcdb.HasPaymentOnDateParams{
		CreditID: row.CreditID, PaymentDate: row.PaymentDate,
	})
	if err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}

	if row.TransactionID != nil {
		return applyPrecreatedPayment(ctx, db, row)
	}

	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	payDate, err := timeutil.ParseUTC(row.PaymentDate)
	if err != nil {
		return false, err
	}
	creditName := row.CreditName
	txID, err := insertCreditExpenseTransaction(ctx, dbTx, row.UserID, row.DebitAccountID, creditName, row.Amount, payDate, true)
	if err != nil {
		return false, err
	}

	n, err := q.ApplyCreditPayment(ctx, sqlcdb.ApplyCreditPaymentParams{
		Kind: "auto", TransactionID: &txID, ID: row.ID, CreditID: row.CreditID,
	})
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}

	creditRow, err := q.GetCreditByID(ctx, sqlcdb.GetCreditByIDParams{ID: row.CreditID, UserID: row.UserID})
	if err != nil {
		return false, err
	}
	newPaid := creditRow.PaidAmount + row.Amount
	nowStr := time.Now().UTC().Format(time.RFC3339)
	if err := q.UpdateCreditPaidAmount(ctx, sqlcdb.UpdateCreditPaidAmountParams{
		PaidAmount: newPaid, UpdatedAt: nowStr, ID: row.CreditID, UserID: row.UserID,
	}); err != nil {
		return false, err
	}
	if err := maybeAutoClose(ctx, dbTx, row.UserID, row.CreditID, creditRow.PrincipalAmount, newPaid, nowStr); err != nil {
		return false, err
	}

	if err := dbTx.Commit(); err != nil {
		return false, err
	}
	syncAccountBalances(ctx, db, row.UserID, row.DebitAccountID)
	return true, nil
}

func applyPrecreatedPayment(ctx context.Context, db *sql.DB, row sqlcdb.ListDueCreditPaymentsRow) (bool, error) {
	dbTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = dbTx.Rollback() }()

	q := queries(dbTx)
	n, err := q.MarkCreditPaymentApplied(ctx, sqlcdb.MarkCreditPaymentAppliedParams{
		ID: row.ID, CreditID: row.CreditID,
	})
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	if row.TransactionID != nil {
		if _, err := q.ActivateTransaction(ctx, sqlcdb.ActivateTransactionParams{
			UpdatedAt: nowStr,
			ID:        *row.TransactionID,
			UserID:    row.UserID,
		}); err != nil {
			return false, err
		}
	}

	creditRow, err := q.GetCreditByID(ctx, sqlcdb.GetCreditByIDParams{ID: row.CreditID, UserID: row.UserID})
	if err != nil {
		return false, err
	}
	newPaid := creditRow.PaidAmount + row.Amount
	if err := q.UpdateCreditPaidAmount(ctx, sqlcdb.UpdateCreditPaidAmountParams{
		PaidAmount: newPaid, UpdatedAt: nowStr, ID: row.CreditID, UserID: row.UserID,
	}); err != nil {
		return false, err
	}
	if err := maybeAutoClose(ctx, dbTx, row.UserID, row.CreditID, creditRow.PrincipalAmount, newPaid, nowStr); err != nil {
		return false, err
	}
	if err := dbTx.Commit(); err != nil {
		return false, err
	}
	syncAccountBalances(ctx, db, row.UserID, row.DebitAccountID)
	return true, nil
}

func insertCreditExpenseTransaction(ctx context.Context, db sqlcdb.DBTX, userID, accountID string, creditName *string, amount int64, payDate time.Time, affectsBalance bool) (string, error) {
	catID, err := categoryseed.CreditCategoryID(ctx, db, userID)
	if err != nil {
		return "", err
	}
	kind, err := resolveTransactionKind(ctx, db, userID, payDate)
	if err != nil {
		return "", err
	}
	desc := creditDescription(creditName)
	id := uuid.NewString()
	nowStr := time.Now().UTC().Format(time.RFC3339)
	txDate := timeutil.FormatUTC(payDate)
	affects := int64(1)
	if !affectsBalance {
		affects = 0
	}
	if err := queries(db).InsertTransaction(ctx, sqlcdb.InsertTransactionParams{
		ID: id, UserID: userID, AccountID: accountID,
		Type: "expense", Kind: kind, Amount: amount, Description: &desc,
		CategoryID: &catID, SubcategoryID: nil,
		TransferGroupID: nil, TransferAccountID: nil,
		TransactionDate: txDate, AffectsBalance: affects,
		CreatedAt: nowStr, UpdatedAt: nowStr,
	}); err != nil {
		return "", err
	}
	return id, nil
}

func creditDescription(name *string) string {
	if name != nil && *name != "" {
		return *name
	}
	return "Кредит"
}

func cancelUnappliedPayments(ctx context.Context, db sqlcdb.DBTX, userID, creditID string) error {
	q := queries(db)
	payments, err := q.ListCreditPayments(ctx, creditID)
	if err != nil {
		return err
	}
	var txIDs []string
	for _, p := range payments {
		if p.IsApplied == 1 {
			continue
		}
		if p.TransactionID != nil {
			txIDs = append(txIDs, *p.TransactionID)
		}
	}
	if _, err := q.DeleteUnappliedCreditPayments(ctx, creditID); err != nil {
		return err
	}
	for _, txID := range txIDs {
		if _, err := q.DeleteTransaction(ctx, sqlcdb.DeleteTransactionParams{ID: txID, UserID: userID}); err != nil {
			return err
		}
	}
	return nil
}

func updateFutureTransactionAccounts(ctx context.Context, db sqlcdb.DBTX, userID, creditID, newAccountID, nowStr string) error {
	payments, err := queries(db).ListCreditPayments(ctx, creditID)
	if err != nil {
		return err
	}
	for _, p := range payments {
		if p.IsApplied == 1 || p.TransactionID == nil {
			continue
		}
		_, err := queries(db).UpdateFutureTransactionAccount(ctx, sqlcdb.UpdateFutureTransactionAccountParams{
			AccountID: newAccountID, UpdatedAt: nowStr, ID: *p.TransactionID, UserID: userID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func maybeAutoClose(ctx context.Context, db sqlcdb.DBTX, userID, creditID string, principal, paid int64, nowStr string) error {
	if paid < principal {
		return nil
	}
	applied, err := queries(db).CountAppliedCreditPayments(ctx, creditID)
	if err != nil {
		return err
	}
	total, err := queries(db).CountCreditPayments(ctx, creditID)
	if err != nil {
		return err
	}
	if applied < total {
		return nil
	}
	closedAt := timeutil.FormatUTC(timeutil.NowUTC())
	_, err = queries(db).CloseCredit(ctx, sqlcdb.CloseCreditParams{
		ClosedAt: &closedAt, UpdatedAt: nowStr, ID: creditID, UserID: userID,
	})
	return err
}

func loadSchedule(ctx context.Context, db *sql.DB, creditID string) ([]CreditPayment, error) {
	rows, err := queries(db).ListCreditPayments(ctx, creditID)
	if err != nil {
		return nil, err
	}
	out := make([]CreditPayment, 0, len(rows))
	for _, r := range rows {
		out = append(out, paymentFromRow(r))
	}
	return out, nil
}

func paymentFromRow(r sqlcdb.ListCreditPaymentsRow) CreditPayment {
	return CreditPayment{
		ID: r.ID, CreditID: r.CreditID, TransactionID: r.TransactionID,
		TransactionKind: r.TransactionKind,
		Amount:          r.Amount, AmountDisplay: money.FormatRubles(r.Amount),
		PaymentDate: r.PaymentDate, Kind: r.Kind,
		IsApplied: r.IsApplied == 1, ExcludeFromStats: r.ExcludeFromStats == 1,
		CreatedAt: r.CreatedAt,
	}
}

// cleanupPaymentDayDuplicates removes legacy early rows and extra unapplied scheduled on the payment day.
func cleanupPaymentDayDuplicates(ctx context.Context, db sqlcdb.DBTX, creditID string, payDate time.Time, userID, keepID string) error {
	tz, err := userTimezoneDBTX(ctx, db, userID)
	if err != nil {
		return err
	}
	q := queries(db)
	payments, err := q.ListCreditPayments(ctx, creditID)
	if err != nil {
		return err
	}
	for _, p := range payments {
		if p.ID == keepID {
			continue
		}
		scheduled, err := timeutil.ParseUTC(p.PaymentDate)
		if err != nil {
			continue
		}
		if !sameCalendarDayInTZ(scheduled, payDate, tz) {
			continue
		}
		remove := p.Kind == "early" || (p.Kind == "scheduled" && p.IsApplied == 0)
		if !remove {
			continue
		}
		if _, err := q.DeleteCreditPaymentByID(ctx, sqlcdb.DeleteCreditPaymentByIDParams{
			ID: p.ID, CreditID: creditID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func sameCalendarDayInTZ(a, b time.Time, tz string) bool {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return false
	}
	aa := a.In(loc)
	bb := b.In(loc)
	return aa.Year() == bb.Year() && aa.Month() == bb.Month() && aa.Day() == bb.Day()
}

func creditFromListRow(ctx context.Context, db *sql.DB, userID string, row sqlcdb.ListCreditsByUserRow) (Credit, error) {
	return enrichCredit(ctx, db, userID, creditFields{
		id: row.ID, name: row.Name, creditKind: row.CreditKind, principal: row.PrincipalAmount,
		propertyPrice: row.PropertyPrice, downPayment: row.DownPayment,
		downPaymentAffectsBalance: row.DownPaymentAffectsBalance, downPaymentTransactionID: row.DownPaymentTransactionID,
		issueDate:  row.IssueDate,
		termMonths: int(row.TermMonths), interestRate: row.InterestRate,
		paymentInterval: row.PaymentInterval, paidAmount: row.PaidAmount,
		monthlyPayment: row.MonthlyPayment, debitAccountID: row.DebitAccountID,
		debitAccountName: row.DebitAccountName, debitTimeLocal: row.DebitTimeLocal,
		bankID: row.BankID, bankName: row.BankName, bankIDLocked: row.BankIDLocked,
		addedRetro: row.AddedRetroactively,
		recordedAt: row.RecordedAt, status: row.Status, closedAt: row.ClosedAt,
		createdAt: row.CreatedAt, updatedAt: row.UpdatedAt,
	})
}

func creditFromGetRow(ctx context.Context, db *sql.DB, userID string, row sqlcdb.GetCreditByIDRow) (Credit, error) {
	return enrichCredit(ctx, db, userID, creditFields{
		id: row.ID, name: row.Name, creditKind: row.CreditKind, principal: row.PrincipalAmount,
		propertyPrice: row.PropertyPrice, downPayment: row.DownPayment,
		downPaymentAffectsBalance: row.DownPaymentAffectsBalance, downPaymentTransactionID: row.DownPaymentTransactionID,
		issueDate:  row.IssueDate,
		termMonths: int(row.TermMonths), interestRate: row.InterestRate,
		paymentInterval: row.PaymentInterval, paidAmount: row.PaidAmount,
		monthlyPayment: row.MonthlyPayment, debitAccountID: row.DebitAccountID,
		debitAccountName: row.DebitAccountName, debitTimeLocal: row.DebitTimeLocal,
		bankID: row.BankID, bankName: row.BankName, bankIDLocked: row.BankIDLocked,
		addedRetro: row.AddedRetroactively,
		recordedAt: row.RecordedAt, status: row.Status, closedAt: row.ClosedAt,
		createdAt: row.CreatedAt, updatedAt: row.UpdatedAt,
	})
}

type creditFields struct {
	id, issueDate, paymentInterval, debitAccountID, debitAccountName string
	recordedAt, status, createdAt, updatedAt                         string
	name, debitTimeLocal, bankID, bankName                           *string
	closedAt                                                         *string
	creditKind                                                       string
	principal, paidAmount, monthlyPayment, downPayment               int64
	propertyPrice                                                    *int64
	downPaymentTransactionID                                         *string
	termMonths                                                       int
	interestRate                                                     float64
	addedRetro, bankIDLocked, downPaymentAffectsBalance              int64
}

func enrichCredit(ctx context.Context, db *sql.DB, userID string, f creditFields) (Credit, error) {
	remaining := RemainingAmount(f.principal, f.paidAmount)
	if f.status == "closed" {
		remaining = 0
	}
	c := Credit{
		ID: f.id, Name: f.name,
		CreditKind:      normalizeCreditKind(f.creditKind),
		PrincipalAmount: f.principal, PrincipalAmountDisplay: money.FormatRubles(f.principal),
		PropertyPrice: f.propertyPrice, DownPayment: f.downPayment, DownPaymentDisplay: money.FormatRubles(f.downPayment),
		DownPaymentAffectsBalance: f.downPaymentAffectsBalance == 1, DownPaymentTransactionID: f.downPaymentTransactionID,
		IssueDate: f.issueDate, TermMonths: f.termMonths, InterestRate: f.interestRate,
		PaymentInterval: f.paymentInterval,
		PaidAmount:      f.paidAmount, PaidAmountDisplay: money.FormatRubles(f.paidAmount),
		MonthlyPayment: f.monthlyPayment, MonthlyPaymentDisplay: money.FormatRubles(f.monthlyPayment),
		RemainingAmount: remaining, RemainingAmountDisplay: money.FormatRubles(remaining),
		DebitAccountID: f.debitAccountID, DebitAccountName: f.debitAccountName,
		DebitTimeLocal: f.debitTimeLocal, BankID: f.bankID, BankName: f.bankName, BankIDLocked: f.bankIDLocked == 1,
		AddedRetroactively: f.addedRetro == 1, RecordedAt: f.recordedAt,
		Status: f.status, ClosedAt: f.closedAt,
		IsInstallment: f.interestRate == 0,
		CreatedAt:     f.createdAt, UpdatedAt: f.updatedAt,
	}
	if f.propertyPrice != nil {
		formatted := money.FormatRubles(*f.propertyPrice)
		c.PropertyPriceDisplay = &formatted
	}

	if f.status == "active" {
		if err := repairMissingSchedule(ctx, db, userID, f); err != nil {
			return Credit{}, err
		}
	}

	next, err := queries(db).ListCreditPayments(ctx, f.id)
	if err != nil {
		return Credit{}, err
	}
	for _, p := range next {
		if p.IsApplied == 0 {
			d := p.PaymentDate
			a := p.Amount
			c.NextPaymentDate = &d
			c.NextPaymentAmount = &a
			break
		}
	}
	if c.NextPaymentDate == nil && c.Status == "active" && c.RemainingAmount > 0 {
		d, a, err := computeFallbackNextPayment(f)
		if err != nil {
			return Credit{}, err
		}
		if d != nil {
			c.NextPaymentDate = d
			c.NextPaymentAmount = a
		}
	}
	return c, nil
}

func validateActiveAccount(ctx context.Context, db *sql.DB, userID, accountID string) error {
	row, err := queries(db).GetAccountByID(ctx, sqlcdb.GetAccountByIDParams{ID: accountID, UserID: userID})
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidAccount
	}
	if err != nil {
		return err
	}
	if row.Status != "active" {
		return ErrAccountArchived
	}
	return nil
}

func validateBank(ctx context.Context, db *sql.DB, bankID *string) error {
	if bankID == nil {
		return nil
	}
	ok, err := queries(db).BankExists(ctx, *bankID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidBank
	}
	return nil
}

func normalizeDebitTimeLocal(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if !localTimePattern.MatchString(trimmed) {
		return nil, ErrInvalidDebitTime
	}
	return &trimmed, nil
}

func normalizeNullableID(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func normalizeCreditKind(kind string) string {
	trimmed := strings.TrimSpace(kind)
	if trimmed == "" {
		return CreditKindConsumer
	}
	return trimmed
}

func isValidCreditKind(kind string) bool {
	return kind == CreditKindConsumer || kind == CreditKindMortgage
}

func normalizeCreditAmounts(kind string, principal int64, propertyPrice *int64, downPayment int64) (int64, *int64, int64, error) {
	if downPayment < 0 {
		return 0, nil, 0, ErrInvalidMortgageFields
	}
	if kind == CreditKindMortgage {
		if propertyPrice == nil || *propertyPrice <= 0 {
			return 0, nil, 0, ErrInvalidMortgageFields
		}
		if downPayment >= *propertyPrice {
			return 0, nil, 0, ErrInvalidMortgageFields
		}
		financed := *propertyPrice - downPayment
		return financed, propertyPrice, downPayment, nil
	}
	if principal <= 0 {
		return principal, nil, 0, nil
	}
	return principal, nil, 0, nil
}

func userTimezone(ctx context.Context, db *sql.DB, userID string) (string, error) {
	tz, err := queries(db).GetUserTimezone(ctx, userID)
	if err != nil {
		return "", err
	}
	if tz == "" {
		return "Europe/Moscow", nil
	}
	return tz, nil
}

func userTimezoneDBTX(ctx context.Context, db sqlcdb.DBTX, userID string) (string, error) {
	tz, err := sqlcdb.New(db).GetUserTimezone(ctx, userID)
	if err != nil {
		return "", err
	}
	if tz == "" {
		return "Europe/Moscow", nil
	}
	return tz, nil
}

func resolveTransactionKind(ctx context.Context, db sqlcdb.DBTX, userID string, txDate time.Time) (string, error) {
	tz, err := userTimezoneDBTX(ctx, db, userID)
	if err != nil {
		return "", err
	}
	future, err := timeutil.IsFutureInTZ(txDate, timeutil.NowUTC(), tz)
	if err != nil {
		return "", err
	}
	if future {
		return "future", nil
	}
	return "manual", nil
}

func syncAccountBalances(ctx context.Context, db *sql.DB, userID string, accountIDs ...string) {
	ids := make([]string, 0, len(accountIDs))
	for _, id := range accountIDs {
		if strings.TrimSpace(id) != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return
	}
	_ = accountbalance.Refresh(ctx, db, userID, ids...)
}

// TodayCutoffUTC returns end-of-today in user TZ as UTC datetime string for due payment queries.
func TodayCutoffUTC(tz string, now time.Time) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return "", fmt.Errorf("invalid timezone: %w", err)
	}
	inTZ := now.In(loc)
	year, month, day := inTZ.Date()
	end := time.Date(year, month, day, 23, 59, 59, 0, loc).UTC()
	return timeutil.FormatUTC(end), nil
}
