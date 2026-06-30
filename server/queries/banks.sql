-- name: BankExists :one
SELECT EXISTS(SELECT 1 FROM banks WHERE id = ?) AS ok;

-- name: ListBanks :many
SELECT id, name, bic, icon_path, sort_order
FROM banks
ORDER BY sort_order, name;

-- name: UpsertBank :exec
INSERT INTO banks (id, name, bic, icon_path, sort_order)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    name = excluded.name,
    bic = excluded.bic,
    icon_path = excluded.icon_path,
    sort_order = excluded.sort_order;

