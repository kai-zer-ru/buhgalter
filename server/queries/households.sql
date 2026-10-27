-- name: GetHouseholdByUserID :one
SELECT h.id, h.name, h.kind, h.created_at, h.owner_user_id, h.principal_user_id, hm.role
FROM household_members hm
JOIN households h ON h.id = hm.household_id
WHERE hm.user_id = ?;

-- name: GetHouseholdByID :one
SELECT id, name, kind, created_at, owner_user_id, principal_user_id
FROM households
WHERE id = ?;

-- name: InsertHousehold :exec
INSERT INTO households (id, name, kind, owner_user_id, principal_user_id, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'));

-- name: UpdateHouseholdName :exec
UPDATE households
SET name = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: UpdateHouseholdOwner :exec
UPDATE households
SET owner_user_id = ?, updated_at = datetime('now')
WHERE id = ?;

-- name: InsertHouseholdMember :exec
INSERT INTO household_members (household_id, user_id, role, joined_at)
VALUES (?, ?, ?, datetime('now'));

-- name: ListHouseholdMembers :many
SELECT hm.user_id, hm.role, hm.joined_at,
       COALESCE(u.display_name, '') AS display_name, u.login
FROM household_members hm
JOIN users u ON u.id = hm.user_id
WHERE hm.household_id = ?
ORDER BY CASE hm.role WHEN 'owner' THEN 0 ELSE 1 END, hm.joined_at ASC;

-- name: ListHouseholdMemberIDs :many
SELECT user_id FROM household_members WHERE household_id = ?;

-- name: CountHouseholdMembers :one
SELECT COUNT(*) FROM household_members WHERE household_id = ?;

-- name: GetHouseholdMember :one
SELECT household_id, user_id, role, joined_at
FROM household_members
WHERE household_id = ? AND user_id = ?;

-- name: DeleteHouseholdMember :execrows
DELETE FROM household_members WHERE household_id = ? AND user_id = ?;

-- name: UpdateHouseholdMemberRole :exec
UPDATE household_members SET role = ? WHERE household_id = ? AND user_id = ?;

-- name: InsertHouseholdInvite :exec
INSERT INTO household_invites (id, household_id, code, status, created_by, created_at, expires_at)
VALUES (?, ?, ?, 'pending', ?, datetime('now'), ?);

-- name: ListPendingHouseholdInvites :many
SELECT id, code, created_at, expires_at
FROM household_invites
WHERE household_id = ? AND status = 'pending' AND expires_at > ?
ORDER BY created_at DESC;

-- name: GetHouseholdInviteByCode :one
SELECT i.id, i.household_id, i.code, i.status, i.created_by, i.created_at, i.expires_at, i.used_by, i.used_at,
       h.name AS household_name, h.owner_user_id, h.principal_user_id
FROM household_invites i
JOIN households h ON h.id = i.household_id
WHERE i.code = ?;

-- name: GetHouseholdInviteByID :one
SELECT id, household_id, code, status, created_by, created_at, expires_at
FROM household_invites
WHERE id = ? AND household_id = ?;

-- name: ConsumeHouseholdInvite :execrows
UPDATE household_invites
SET status = 'used', used_by = ?, used_at = datetime('now')
WHERE id = ? AND status = 'pending' AND expires_at > ?;

-- name: RevokeHouseholdInvite :execrows
UPDATE household_invites
SET status = 'revoked'
WHERE id = ? AND household_id = ? AND status = 'pending';

-- name: DeletePendingHouseholdInvites :exec
DELETE FROM household_invites WHERE household_id = ? AND status = 'pending';

-- name: CountNonDeletedAccountsByUser :one
SELECT COUNT(*) FROM accounts WHERE user_id = ? AND status != 'deleted';

-- name: ListHouseholdMemberIDsByPrincipal :many
SELECT hm.user_id
FROM household_members hm
JOIN households h ON h.id = hm.household_id
WHERE h.principal_user_id = ?;

-- name: DeleteHouseholdMembersByHousehold :exec
DELETE FROM household_members WHERE household_id = ?;

-- name: DeleteHousehold :exec
DELETE FROM households WHERE id = ?;

-- name: ReassignInviteCreatedBy :exec
UPDATE household_invites SET created_by = ?
WHERE household_id = ? AND created_by = ?;
