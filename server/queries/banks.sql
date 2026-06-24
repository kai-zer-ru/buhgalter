-- name: BankExists :one
SELECT EXISTS(SELECT 1 FROM banks WHERE id = ?) AS ok;

-- name: ListBanks :many
SELECT id, name, bic, icon_path, sort_order
FROM banks
ORDER BY sort_order, name;
