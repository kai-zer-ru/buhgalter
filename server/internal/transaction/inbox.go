package transaction

import (
	"context"
	"database/sql"

	sqlcdb "github.com/kai-zer-ru/buhgalter/internal/db/sqlc"
)

func CountInbox(ctx context.Context, db *sql.DB, userID string) (int64, error) {
	n, err := queries(db).CountInboxTransactions(ctx, userID)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func ListInbox(ctx context.Context, db *sql.DB, userID string, page, limit int, sort string) (ListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := int64((page - 1) * limit)
	lim := int64(limit)
	q := queries(db)
	total, err := q.CountInboxTransactions(ctx, userID)
	if err != nil {
		return ListResult{}, err
	}
	var items []Transaction
	if sort == "date_asc" {
		rows, err := q.ListInboxTransactionsDateAsc(ctx, sqlcdb.ListInboxTransactionsDateAscParams{
			UserID: userID,
			Limit:  lim,
			Offset: offset,
		})
		if err != nil {
			return ListResult{}, err
		}
		items = make([]Transaction, 0, len(rows))
		for _, row := range rows {
			items = append(items, txFromInboxAsc(row))
		}
	} else {
		rows, err := q.ListInboxTransactionsDateDesc(ctx, sqlcdb.ListInboxTransactionsDateDescParams{
			UserID: userID,
			Limit:  lim,
			Offset: offset,
		})
		if err != nil {
			return ListResult{}, err
		}
		items = make([]Transaction, 0, len(rows))
		for _, row := range rows {
			items = append(items, txFromInboxDesc(row))
		}
	}
	if items == nil {
		items = []Transaction{}
	}
	if err := attachTags(ctx, db, items); err != nil {
		return ListResult{}, err
	}
	return ListResult{
		Data: items,
		Meta: ListMeta{Page: page, Limit: limit, Total: total},
	}, nil
}
