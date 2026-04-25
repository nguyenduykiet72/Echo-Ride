-- name: CreateIdentity :one
INSERT INTO t_identities (
    identity_email,
    identity_phone,
    identity_password_hash,
    identity_role
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetIdentityByEmail :one
SELECT * FROM t_identities
WHERE identity_email = $1
LIMIT 1;

-- name: GetIdentityByPhone :one
SELECT * FROM t_identities
WHERE identity_phone = $1
LIMIT 1;

-- name: UpdateIdentityStatus :one
UPDATE t_identities
SET identity_status = $2, identity_updated_at = NOW()
WHERE identity_id = $1
RETURNING *;