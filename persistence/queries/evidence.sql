-- name: InsertEvidence :one
INSERT INTO evidence (id, content_type, content_hash, payload_ref, labels, ingested_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (content_hash) DO NOTHING
RETURNING id, content_hash, leaf_index;

-- name: GetEvidenceByID :one
SELECT id, content_type, content_hash, payload_ref, labels, ingested_at, ingested_by, leaf_index
FROM evidence WHERE id = $1;

-- name: InsertLeaf :one
INSERT INTO tree_leaves (leaf_index, leaf_hash) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: InsertSTH :one
INSERT INTO signed_tree_heads (tree_size, root_hash, key_id, signature)
VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING RETURNING id;

-- name: LatestSTH :one
SELECT tree_size, root_hash, key_id, signature, published_at
FROM signed_tree_heads ORDER BY tree_size DESC LIMIT 1;
