// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: universe.sql

package sqlc

import (
	"context"
	"database/sql"
	"time"
)

const deleteFederationProofSyncLog = `-- name: DeleteFederationProofSyncLog :exec
WITH selected_server_id AS (
    -- Select the server ids from the universe_servers table for the specified
    -- hosts.
    SELECT id
    FROM universe_servers
    WHERE
        (server_host = $4
            OR $4 IS NULL)
)
DELETE FROM federation_proof_sync_log
WHERE
    servers_id IN (SELECT id FROM selected_server_id) AND
    (status = $1
        OR $1 IS NULL) AND
    (timestamp >= $2
        OR $2 IS NULL) AND
    (attempt_counter >= $3
        OR $3 IS NULL)
`

type DeleteFederationProofSyncLogParams struct {
	Status            sql.NullString
	MinTimestamp      sql.NullTime
	MinAttemptCounter sql.NullInt64
	ServerHost        sql.NullString
}

func (q *Queries) DeleteFederationProofSyncLog(ctx context.Context, arg DeleteFederationProofSyncLogParams) error {
	_, err := q.db.ExecContext(ctx, deleteFederationProofSyncLog,
		arg.Status,
		arg.MinTimestamp,
		arg.MinAttemptCounter,
		arg.ServerHost,
	)
	return err
}

const deleteMultiverseLeaf = `-- name: DeleteMultiverseLeaf :exec
DELETE FROM multiverse_leaves
WHERE leaf_node_namespace = $1 AND leaf_node_key = $2
`

type DeleteMultiverseLeafParams struct {
	Namespace   string
	LeafNodeKey []byte
}

func (q *Queries) DeleteMultiverseLeaf(ctx context.Context, arg DeleteMultiverseLeafParams) error {
	_, err := q.db.ExecContext(ctx, deleteMultiverseLeaf, arg.Namespace, arg.LeafNodeKey)
	return err
}

const deleteUniverseEvents = `-- name: DeleteUniverseEvents :exec
WITH root_id AS (
    SELECT id
    FROM universe_roots
    WHERE namespace_root = $1
)
DELETE FROM universe_events
WHERE universe_root_id = (SELECT id FROM root_id)
`

func (q *Queries) DeleteUniverseEvents(ctx context.Context, namespaceRoot string) error {
	_, err := q.db.ExecContext(ctx, deleteUniverseEvents, namespaceRoot)
	return err
}

const deleteUniverseLeaves = `-- name: DeleteUniverseLeaves :exec
DELETE FROM universe_leaves
WHERE leaf_node_namespace = $1
`

func (q *Queries) DeleteUniverseLeaves(ctx context.Context, namespace string) error {
	_, err := q.db.ExecContext(ctx, deleteUniverseLeaves, namespace)
	return err
}

const deleteUniverseRoot = `-- name: DeleteUniverseRoot :exec
DELETE FROM universe_roots
WHERE namespace_root = $1
`

func (q *Queries) DeleteUniverseRoot(ctx context.Context, namespaceRoot string) error {
	_, err := q.db.ExecContext(ctx, deleteUniverseRoot, namespaceRoot)
	return err
}

const deleteUniverseServer = `-- name: DeleteUniverseServer :exec
DELETE FROM universe_servers
WHERE server_host = $1 OR id = $2
`

type DeleteUniverseServerParams struct {
	TargetServer string
	TargetID     int64
}

func (q *Queries) DeleteUniverseServer(ctx context.Context, arg DeleteUniverseServerParams) error {
	_, err := q.db.ExecContext(ctx, deleteUniverseServer, arg.TargetServer, arg.TargetID)
	return err
}

const fetchMultiverseRoot = `-- name: FetchMultiverseRoot :one
SELECT proof_type, n.hash_key as multiverse_root_hash, n.sum as multiverse_root_sum
FROM multiverse_roots r
JOIN mssmt_roots m
    ON r.namespace_root = m.namespace
JOIN mssmt_nodes n
    ON m.root_hash = n.hash_key AND
       m.namespace = n.namespace
WHERE namespace_root = $1
`

type FetchMultiverseRootRow struct {
	ProofType          string
	MultiverseRootHash []byte
	MultiverseRootSum  int64
}

func (q *Queries) FetchMultiverseRoot(ctx context.Context, namespaceRoot string) (FetchMultiverseRootRow, error) {
	row := q.db.QueryRowContext(ctx, fetchMultiverseRoot, namespaceRoot)
	var i FetchMultiverseRootRow
	err := row.Scan(&i.ProofType, &i.MultiverseRootHash, &i.MultiverseRootSum)
	return i, err
}

const fetchUniverseKeys = `-- name: FetchUniverseKeys :many
SELECT leaves.minting_point, leaves.script_key_bytes
FROM universe_leaves AS leaves
WHERE leaves.leaf_node_namespace = $1
ORDER BY 
    CASE WHEN $2 = 0 THEN leaves.id END ASC,
    CASE WHEN $2 = 1 THEN leaves.id END DESC
LIMIT $4 OFFSET $3
`

type FetchUniverseKeysParams struct {
	Namespace     string
	SortDirection interface{}
	NumOffset     int32
	NumLimit      int32
}

type FetchUniverseKeysRow struct {
	MintingPoint   []byte
	ScriptKeyBytes []byte
}

func (q *Queries) FetchUniverseKeys(ctx context.Context, arg FetchUniverseKeysParams) ([]FetchUniverseKeysRow, error) {
	rows, err := q.db.QueryContext(ctx, fetchUniverseKeys,
		arg.Namespace,
		arg.SortDirection,
		arg.NumOffset,
		arg.NumLimit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FetchUniverseKeysRow
	for rows.Next() {
		var i FetchUniverseKeysRow
		if err := rows.Scan(&i.MintingPoint, &i.ScriptKeyBytes); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const fetchUniverseRoot = `-- name: FetchUniverseRoot :one
SELECT universe_roots.asset_id, group_key, proof_type,
       mssmt_nodes.hash_key root_hash, mssmt_nodes.sum root_sum,
       genesis_assets.asset_tag asset_name
FROM universe_roots
JOIN mssmt_roots 
    ON universe_roots.namespace_root = mssmt_roots.namespace
JOIN mssmt_nodes 
    ON mssmt_nodes.hash_key = mssmt_roots.root_hash 
       AND mssmt_nodes.namespace = mssmt_roots.namespace
JOIN genesis_assets
    ON genesis_assets.asset_id = universe_roots.asset_id
WHERE mssmt_nodes.namespace = $1
`

type FetchUniverseRootRow struct {
	AssetID   []byte
	GroupKey  []byte
	ProofType string
	RootHash  []byte
	RootSum   int64
	AssetName string
}

func (q *Queries) FetchUniverseRoot(ctx context.Context, namespace string) (FetchUniverseRootRow, error) {
	row := q.db.QueryRowContext(ctx, fetchUniverseRoot, namespace)
	var i FetchUniverseRootRow
	err := row.Scan(
		&i.AssetID,
		&i.GroupKey,
		&i.ProofType,
		&i.RootHash,
		&i.RootSum,
		&i.AssetName,
	)
	return i, err
}

const insertNewProofEvent = `-- name: InsertNewProofEvent :exec
WITH group_key_root_id AS (
    SELECT id
    FROM universe_roots roots
    WHERE group_key = $1
        AND roots.proof_type = $4
), asset_id_root_id AS (
    SELECT leaves.universe_root_id AS id
    FROM universe_leaves leaves
    JOIN universe_roots roots
        ON leaves.universe_root_id = roots.id
    JOIN genesis_info_view gen
        ON leaves.asset_genesis_id = gen.gen_asset_id
    WHERE gen.asset_id = $5
        AND roots.proof_type = $4
    LIMIT 1
)
INSERT INTO universe_events (
    event_type, universe_root_id, event_time, event_timestamp
) VALUES (
    'NEW_PROOF',
        CASE WHEN length($1) > 0 THEN (
            SELECT id FROM group_key_root_id
        ) ELSE (
            SELECT id FROM asset_id_root_id
        ) END,
    $2, $3
)
`

type InsertNewProofEventParams struct {
	GroupKeyXOnly  interface{}
	EventTime      time.Time
	EventTimestamp int64
	ProofType      string
	AssetID        []byte
}

func (q *Queries) InsertNewProofEvent(ctx context.Context, arg InsertNewProofEventParams) error {
	_, err := q.db.ExecContext(ctx, insertNewProofEvent,
		arg.GroupKeyXOnly,
		arg.EventTime,
		arg.EventTimestamp,
		arg.ProofType,
		arg.AssetID,
	)
	return err
}

const insertNewSyncEvent = `-- name: InsertNewSyncEvent :exec
WITH group_key_root_id AS (
    SELECT id
    FROM universe_roots roots
    WHERE group_key = $1
      AND roots.proof_type = $4
), asset_id_root_id AS (
    SELECT leaves.universe_root_id AS id
    FROM universe_leaves leaves
    JOIN universe_roots roots
        ON leaves.universe_root_id = roots.id
    JOIN genesis_info_view gen
        ON leaves.asset_genesis_id = gen.gen_asset_id
    WHERE gen.asset_id = $5
        AND roots.proof_type = $4
    LIMIT 1
)
INSERT INTO universe_events (
    event_type, universe_root_id, event_time, event_timestamp
) VALUES (
    'SYNC',
        CASE WHEN length($1) > 0 THEN (
            SELECT id FROM group_key_root_id
        ) ELSE (
            SELECT id FROM asset_id_root_id
        ) END,
    $2, $3
)
`

type InsertNewSyncEventParams struct {
	GroupKeyXOnly  interface{}
	EventTime      time.Time
	EventTimestamp int64
	ProofType      string
	AssetID        []byte
}

func (q *Queries) InsertNewSyncEvent(ctx context.Context, arg InsertNewSyncEventParams) error {
	_, err := q.db.ExecContext(ctx, insertNewSyncEvent,
		arg.GroupKeyXOnly,
		arg.EventTime,
		arg.EventTimestamp,
		arg.ProofType,
		arg.AssetID,
	)
	return err
}

const insertUniverseServer = `-- name: InsertUniverseServer :exec
INSERT INTO universe_servers(
    server_host, last_sync_time
) VALUES (
    $1, $2
)
`

type InsertUniverseServerParams struct {
	ServerHost   string
	LastSyncTime time.Time
}

func (q *Queries) InsertUniverseServer(ctx context.Context, arg InsertUniverseServerParams) error {
	_, err := q.db.ExecContext(ctx, insertUniverseServer, arg.ServerHost, arg.LastSyncTime)
	return err
}

const logServerSync = `-- name: LogServerSync :exec
UPDATE universe_servers
SET last_sync_time = $1
WHERE server_host = $2
`

type LogServerSyncParams struct {
	NewSyncTime  time.Time
	TargetServer string
}

func (q *Queries) LogServerSync(ctx context.Context, arg LogServerSyncParams) error {
	_, err := q.db.ExecContext(ctx, logServerSync, arg.NewSyncTime, arg.TargetServer)
	return err
}

const queryAssetStatsPerDayPostgres = `-- name: QueryAssetStatsPerDayPostgres :many
SELECT
    to_char(to_timestamp(event_timestamp), 'YYYY-MM-DD') AS day,
    SUM(CASE WHEN event_type = 'SYNC' THEN 1 ELSE 0 END) AS sync_events,
    SUM(CASE WHEN event_type = 'NEW_PROOF' THEN 1 ELSE 0 END) AS new_proof_events
FROM universe_events
WHERE event_type IN ('SYNC', 'NEW_PROOF') 
      AND event_timestamp BETWEEN $1 AND $2
GROUP BY day
ORDER BY day
`

type QueryAssetStatsPerDayPostgresParams struct {
	StartTime int64
	EndTime   int64
}

type QueryAssetStatsPerDayPostgresRow struct {
	Day            string
	SyncEvents     int64
	NewProofEvents int64
}

func (q *Queries) QueryAssetStatsPerDayPostgres(ctx context.Context, arg QueryAssetStatsPerDayPostgresParams) ([]QueryAssetStatsPerDayPostgresRow, error) {
	rows, err := q.db.QueryContext(ctx, queryAssetStatsPerDayPostgres, arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QueryAssetStatsPerDayPostgresRow
	for rows.Next() {
		var i QueryAssetStatsPerDayPostgresRow
		if err := rows.Scan(&i.Day, &i.SyncEvents, &i.NewProofEvents); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queryAssetStatsPerDaySqlite = `-- name: QueryAssetStatsPerDaySqlite :many
SELECT
    cast(strftime('%Y-%m-%d', datetime(event_timestamp, 'unixepoch')) as text) AS day,
    SUM(CASE WHEN event_type = 'SYNC' THEN 1 ELSE 0 END) AS sync_events,
    SUM(CASE WHEN event_type = 'NEW_PROOF' THEN 1 ELSE 0 END) AS new_proof_events
FROM universe_events
WHERE event_type IN ('SYNC', 'NEW_PROOF') AND
      event_timestamp >= $1 AND event_timestamp <= $2
GROUP BY day
ORDER BY day
`

type QueryAssetStatsPerDaySqliteParams struct {
	StartTime int64
	EndTime   int64
}

type QueryAssetStatsPerDaySqliteRow struct {
	Day            string
	SyncEvents     int64
	NewProofEvents int64
}

func (q *Queries) QueryAssetStatsPerDaySqlite(ctx context.Context, arg QueryAssetStatsPerDaySqliteParams) ([]QueryAssetStatsPerDaySqliteRow, error) {
	rows, err := q.db.QueryContext(ctx, queryAssetStatsPerDaySqlite, arg.StartTime, arg.EndTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QueryAssetStatsPerDaySqliteRow
	for rows.Next() {
		var i QueryAssetStatsPerDaySqliteRow
		if err := rows.Scan(&i.Day, &i.SyncEvents, &i.NewProofEvents); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queryFederationGlobalSyncConfigs = `-- name: QueryFederationGlobalSyncConfigs :many
SELECT proof_type, allow_sync_insert, allow_sync_export
FROM federation_global_sync_config
ORDER BY proof_type
`

func (q *Queries) QueryFederationGlobalSyncConfigs(ctx context.Context) ([]FederationGlobalSyncConfig, error) {
	rows, err := q.db.QueryContext(ctx, queryFederationGlobalSyncConfigs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FederationGlobalSyncConfig
	for rows.Next() {
		var i FederationGlobalSyncConfig
		if err := rows.Scan(&i.ProofType, &i.AllowSyncInsert, &i.AllowSyncExport); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queryFederationProofSyncLog = `-- name: QueryFederationProofSyncLog :many
SELECT
    log.id, status, timestamp, sync_direction, attempt_counter,
    -- Select fields from the universe_servers table.
    server.id AS server_id,
    server.server_host,
    -- Select universe leaf related fields.
    leaf.minting_point AS leaf_minting_point_bytes,
    leaf.script_key_bytes AS leaf_script_key_bytes,
    mssmt_node.value AS leaf_genesis_proof,
    genesis.gen_asset_id AS leaf_gen_asset_id,
    genesis.asset_id AS leaf_asset_id,
    -- Select fields from the universe_roots table.
    root.asset_id AS uni_asset_id,
    root.group_key AS uni_group_key,
    root.proof_type AS uni_proof_type
FROM federation_proof_sync_log AS log
JOIN universe_leaves AS leaf
    ON leaf.id = log.proof_leaf_id
JOIN mssmt_nodes AS mssmt_node
     ON leaf.leaf_node_key = mssmt_node.key 
        AND leaf.leaf_node_namespace = mssmt_node.namespace
JOIN genesis_info_view AS genesis
     ON leaf.asset_genesis_id = genesis.gen_asset_id
JOIN universe_servers AS server
    ON server.id = log.servers_id
JOIN universe_roots AS root
    ON root.id = log.universe_root_id
WHERE (log.sync_direction = $1 OR $1 IS NULL)
      AND (log.status = $2 OR $2 IS NULL)
      -- Universe leaves WHERE clauses.
      AND (leaf.leaf_node_namespace = $3 OR $3 IS NULL)
      AND (leaf.minting_point = $4 OR $4 IS NULL)
      AND (leaf.script_key_bytes = $5 OR $5 IS NULL)
`

type QueryFederationProofSyncLogParams struct {
	SyncDirection         sql.NullString
	Status                sql.NullString
	LeafNamespace         sql.NullString
	LeafMintingPointBytes []byte
	LeafScriptKeyBytes    []byte
}

type QueryFederationProofSyncLogRow struct {
	ID                    int64
	Status                string
	Timestamp             time.Time
	SyncDirection         string
	AttemptCounter        int64
	ServerID              int64
	ServerHost            string
	LeafMintingPointBytes []byte
	LeafScriptKeyBytes    []byte
	LeafGenesisProof      []byte
	LeafGenAssetID        int64
	LeafAssetID           []byte
	UniAssetID            []byte
	UniGroupKey           []byte
	UniProofType          string
}

// Join on mssmt_nodes to get leaf related fields.
// Join on genesis_info_view to get leaf related fields.
func (q *Queries) QueryFederationProofSyncLog(ctx context.Context, arg QueryFederationProofSyncLogParams) ([]QueryFederationProofSyncLogRow, error) {
	rows, err := q.db.QueryContext(ctx, queryFederationProofSyncLog,
		arg.SyncDirection,
		arg.Status,
		arg.LeafNamespace,
		arg.LeafMintingPointBytes,
		arg.LeafScriptKeyBytes,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QueryFederationProofSyncLogRow
	for rows.Next() {
		var i QueryFederationProofSyncLogRow
		if err := rows.Scan(
			&i.ID,
			&i.Status,
			&i.Timestamp,
			&i.SyncDirection,
			&i.AttemptCounter,
			&i.ServerID,
			&i.ServerHost,
			&i.LeafMintingPointBytes,
			&i.LeafScriptKeyBytes,
			&i.LeafGenesisProof,
			&i.LeafGenAssetID,
			&i.LeafAssetID,
			&i.UniAssetID,
			&i.UniGroupKey,
			&i.UniProofType,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queryFederationUniSyncConfigs = `-- name: QueryFederationUniSyncConfigs :many
SELECT namespace, asset_id, group_key, proof_type, allow_sync_insert, allow_sync_export
FROM federation_uni_sync_config
ORDER BY group_key NULLS LAST, asset_id NULLS LAST, proof_type
`

func (q *Queries) QueryFederationUniSyncConfigs(ctx context.Context) ([]FederationUniSyncConfig, error) {
	rows, err := q.db.QueryContext(ctx, queryFederationUniSyncConfigs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FederationUniSyncConfig
	for rows.Next() {
		var i FederationUniSyncConfig
		if err := rows.Scan(
			&i.Namespace,
			&i.AssetID,
			&i.GroupKey,
			&i.ProofType,
			&i.AllowSyncInsert,
			&i.AllowSyncExport,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queryMultiverseLeaves = `-- name: QueryMultiverseLeaves :many
SELECT r.namespace_root, r.proof_type, l.asset_id, l.group_key, 
       smt_nodes.value AS universe_root_hash, smt_nodes.sum AS universe_root_sum
FROM multiverse_leaves l
JOIN mssmt_nodes smt_nodes
  ON l.leaf_node_key = smt_nodes.key AND
     l.leaf_node_namespace = smt_nodes.namespace
JOIN multiverse_roots r
  ON l.multiverse_root_id = r.id
WHERE r.proof_type = $1 AND
      (l.asset_id = $2 OR $2 IS NULL) AND
      (l.group_key = $3 OR $3 IS NULL)
`

type QueryMultiverseLeavesParams struct {
	ProofType string
	AssetID   []byte
	GroupKey  []byte
}

type QueryMultiverseLeavesRow struct {
	NamespaceRoot    string
	ProofType        string
	AssetID          []byte
	GroupKey         []byte
	UniverseRootHash []byte
	UniverseRootSum  int64
}

func (q *Queries) QueryMultiverseLeaves(ctx context.Context, arg QueryMultiverseLeavesParams) ([]QueryMultiverseLeavesRow, error) {
	rows, err := q.db.QueryContext(ctx, queryMultiverseLeaves, arg.ProofType, arg.AssetID, arg.GroupKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QueryMultiverseLeavesRow
	for rows.Next() {
		var i QueryMultiverseLeavesRow
		if err := rows.Scan(
			&i.NamespaceRoot,
			&i.ProofType,
			&i.AssetID,
			&i.GroupKey,
			&i.UniverseRootHash,
			&i.UniverseRootSum,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queryUniverseAssetStats = `-- name: QueryUniverseAssetStats :many

WITH asset_supply AS (
    SELECT SUM(nodes.sum) AS supply, gen.asset_id AS asset_id
    FROM universe_leaves leaves
    JOIN universe_roots roots
        ON leaves.universe_root_id = roots.id
    JOIN mssmt_nodes nodes
        ON leaves.leaf_node_key = nodes.key AND
           leaves.leaf_node_namespace = nodes.namespace
    JOIN genesis_info_view gen
        ON leaves.asset_genesis_id = gen.gen_asset_id
    WHERE roots.proof_type = 'issuance'
    GROUP BY gen.asset_id
), group_supply AS (
    SELECT sum AS num_assets, uroots.group_key AS group_key
    FROM mssmt_nodes nodes
    JOIN mssmt_roots roots
      ON nodes.hash_key = roots.root_hash AND
         nodes.namespace = roots.namespace
    JOIN universe_roots uroots
      ON roots.namespace = uroots.namespace_root
    WHERE uroots.proof_type = 'issuance'
), asset_info AS (
    SELECT asset_supply.supply, group_supply.num_assets AS group_supply,
           gen.asset_id AS asset_id, 
           gen.asset_tag AS asset_name, gen.asset_type AS asset_type,
           gen.block_height AS genesis_height, gen.prev_out AS genesis_prev_out,
           group_info.tweaked_group_key AS group_key,
           gen.output_index AS anchor_index, gen.anchor_txid AS anchor_txid
    FROM genesis_info_view gen
    JOIN asset_supply
        ON asset_supply.asset_id = gen.asset_id
    -- We use a LEFT JOIN here as not every asset has a group key, so this'll
    -- generate rows that have NULL values for the group key fields if an asset
    -- doesn't have a group key.
    LEFT JOIN key_group_info_view group_info
        ON gen.gen_asset_id = group_info.gen_asset_id
    LEFT JOIN group_supply
        ON group_supply.group_key = group_info.x_only_group_key
    WHERE (gen.asset_tag = $5 OR $5 IS NULL) AND
          (gen.asset_type = $6 OR $6 IS NULL) AND
          (gen.asset_id = $7 OR $7 IS NULL)
)
SELECT asset_info.supply AS asset_supply,
    asset_info.group_supply AS group_supply,
    asset_info.asset_name AS asset_name,
    asset_info.asset_type AS asset_type, asset_info.asset_id AS asset_id,
    asset_info.genesis_height AS genesis_height,
    asset_info.genesis_prev_out AS genesis_prev_out,
    asset_info.group_key AS group_key,
    asset_info.anchor_index AS anchor_index,
    asset_info.anchor_txid AS anchor_txid,
    universe_stats.total_asset_syncs AS total_syncs,
    universe_stats.total_asset_proofs AS total_proofs
FROM asset_info
JOIN universe_stats
    ON asset_info.asset_id = universe_stats.asset_id
WHERE universe_stats.proof_type = 'issuance'
ORDER BY
    CASE WHEN $1 = 'asset_id' AND $2 = 0 THEN
             asset_info.asset_id END ASC,
    CASE WHEN $1 = 'asset_id' AND $2 = 1 THEN
             asset_info.asset_id END DESC,
    CASE WHEN $1 = 'asset_name' AND $2 = 0 THEN
             asset_info.asset_name END ASC ,
    CASE WHEN $1 = 'asset_name' AND $2 = 1 THEN
             asset_info.asset_name END DESC ,
    CASE WHEN $1 = 'asset_type' AND $2 = 0 THEN
             asset_info.asset_type END ASC ,
    CASE WHEN $1 = 'asset_type' AND $2 = 1 THEN
             asset_info.asset_type END DESC,
    CASE WHEN $1 = 'total_syncs' AND $2 = 0 THEN
             universe_stats.total_asset_syncs END ASC ,
    CASE WHEN $1 = 'total_syncs' AND $2 = 1 THEN
             universe_stats.total_asset_syncs END DESC,
    CASE WHEN $1 = 'total_proofs' AND $2 = 0 THEN
             universe_stats.total_asset_proofs END ASC ,
    CASE WHEN $1 = 'total_proofs' AND $2 = 1 THEN
             universe_stats.total_asset_proofs END DESC,
    CASE WHEN $1 = 'genesis_height' AND $2 = 0 THEN
             asset_info.genesis_height END ASC ,
    CASE WHEN $1 = 'genesis_height' AND $2 = 1 THEN
             asset_info.genesis_height END DESC,
    CASE WHEN $1 = 'total_supply' AND $2 = 0 THEN
             asset_info.supply END ASC ,
    CASE WHEN $1 = 'total_supply' AND $2 = 1 THEN
             asset_info.supply END DESC
LIMIT $4 OFFSET $3
`

type QueryUniverseAssetStatsParams struct {
	SortBy        interface{}
	SortDirection interface{}
	NumOffset     int32
	NumLimit      int32
	AssetName     sql.NullString
	AssetType     sql.NullInt16
	AssetID       []byte
}

type QueryUniverseAssetStatsRow struct {
	AssetSupply    int64
	GroupSupply    sql.NullInt64
	AssetName      string
	AssetType      int16
	AssetID        []byte
	GenesisHeight  sql.NullInt32
	GenesisPrevOut []byte
	GroupKey       []byte
	AnchorIndex    int32
	AnchorTxid     []byte
	TotalSyncs     int64
	TotalProofs    int64
}

// TODO(roasbeef): use the universe id instead for the grouping? so namespace
// root, simplifies queries
func (q *Queries) QueryUniverseAssetStats(ctx context.Context, arg QueryUniverseAssetStatsParams) ([]QueryUniverseAssetStatsRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUniverseAssetStats,
		arg.SortBy,
		arg.SortDirection,
		arg.NumOffset,
		arg.NumLimit,
		arg.AssetName,
		arg.AssetType,
		arg.AssetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QueryUniverseAssetStatsRow
	for rows.Next() {
		var i QueryUniverseAssetStatsRow
		if err := rows.Scan(
			&i.AssetSupply,
			&i.GroupSupply,
			&i.AssetName,
			&i.AssetType,
			&i.AssetID,
			&i.GenesisHeight,
			&i.GenesisPrevOut,
			&i.GroupKey,
			&i.AnchorIndex,
			&i.AnchorTxid,
			&i.TotalSyncs,
			&i.TotalProofs,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queryUniverseLeaves = `-- name: QueryUniverseLeaves :many
SELECT leaves.script_key_bytes, gen.gen_asset_id, nodes.value AS genesis_proof, 
       nodes.sum AS sum_amt, gen.asset_id
FROM universe_leaves AS leaves
JOIN mssmt_nodes AS nodes
    ON leaves.leaf_node_key = nodes.key 
       AND leaves.leaf_node_namespace = nodes.namespace
JOIN genesis_info_view AS gen
    ON leaves.asset_genesis_id = gen.gen_asset_id
WHERE leaves.leaf_node_namespace = $1 
      AND (leaves.minting_point = $2 OR 
           $2 IS NULL) 
      AND (leaves.script_key_bytes = $3 OR 
           $3 IS NULL)
`

type QueryUniverseLeavesParams struct {
	Namespace         string
	MintingPointBytes []byte
	ScriptKeyBytes    []byte
}

type QueryUniverseLeavesRow struct {
	ScriptKeyBytes []byte
	GenAssetID     int64
	GenesisProof   []byte
	SumAmt         int64
	AssetID        []byte
}

func (q *Queries) QueryUniverseLeaves(ctx context.Context, arg QueryUniverseLeavesParams) ([]QueryUniverseLeavesRow, error) {
	rows, err := q.db.QueryContext(ctx, queryUniverseLeaves, arg.Namespace, arg.MintingPointBytes, arg.ScriptKeyBytes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []QueryUniverseLeavesRow
	for rows.Next() {
		var i QueryUniverseLeavesRow
		if err := rows.Scan(
			&i.ScriptKeyBytes,
			&i.GenAssetID,
			&i.GenesisProof,
			&i.SumAmt,
			&i.AssetID,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queryUniverseServers = `-- name: QueryUniverseServers :many
SELECT id, server_host, last_sync_time FROM universe_servers
WHERE (id = $1 OR $1 IS NULL) AND
      (server_host = $2
           OR $2 IS NULL)
`

type QueryUniverseServersParams struct {
	ID         sql.NullInt64
	ServerHost sql.NullString
}

func (q *Queries) QueryUniverseServers(ctx context.Context, arg QueryUniverseServersParams) ([]UniverseServer, error) {
	rows, err := q.db.QueryContext(ctx, queryUniverseServers, arg.ID, arg.ServerHost)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []UniverseServer
	for rows.Next() {
		var i UniverseServer
		if err := rows.Scan(&i.ID, &i.ServerHost, &i.LastSyncTime); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const queryUniverseStats = `-- name: QueryUniverseStats :one
WITH stats AS (
    SELECT total_asset_syncs, total_asset_proofs
    FROM universe_stats
), group_ids AS (
    SELECT id
    FROM universe_roots
    WHERE group_key IS NOT NULL
), asset_keys AS (
    SELECT hash_key
    FROM mssmt_nodes nodes
    JOIN mssmt_roots roots
      ON nodes.hash_key = roots.root_hash AND
         nodes.namespace = roots.namespace
    JOIN universe_roots uroots
      ON roots.namespace = uroots.namespace_root
), aggregated AS (
    SELECT COALESCE(SUM(stats.total_asset_syncs), 0) AS total_syncs,
           COALESCE(SUM(stats.total_asset_proofs), 0) AS total_proofs,
           0 AS total_num_groups,
           0 AS total_num_assets
    FROM stats
    UNION ALL
    SELECT 0 AS total_syncs,
           0 AS total_proofs,
           COALESCE(COUNT(group_ids.id), 0) AS total_num_groups,
           0 AS total_num_assets
    FROM group_ids
    UNION ALL
    SELECT 0 AS total_syncs,
           0 AS total_proofs,
           0 AS total_num_groups,
           COALESCE(COUNT(asset_keys.hash_key), 0) AS total_num_assets
    FROM asset_keys
)
SELECT SUM(total_syncs) AS total_syncs,
       SUM(total_proofs) AS total_proofs,
       SUM(total_num_groups) AS total_num_groups,
       SUM(total_num_assets) AS total_num_assets
FROM aggregated
`

type QueryUniverseStatsRow struct {
	TotalSyncs     int64
	TotalProofs    int64
	TotalNumGroups int64
	TotalNumAssets int64
}

func (q *Queries) QueryUniverseStats(ctx context.Context) (QueryUniverseStatsRow, error) {
	row := q.db.QueryRowContext(ctx, queryUniverseStats)
	var i QueryUniverseStatsRow
	err := row.Scan(
		&i.TotalSyncs,
		&i.TotalProofs,
		&i.TotalNumGroups,
		&i.TotalNumAssets,
	)
	return i, err
}

const universeLeaves = `-- name: UniverseLeaves :many
SELECT id, asset_genesis_id, minting_point, script_key_bytes, universe_root_id, leaf_node_key, leaf_node_namespace FROM universe_leaves
`

func (q *Queries) UniverseLeaves(ctx context.Context) ([]UniverseLeafe, error) {
	rows, err := q.db.QueryContext(ctx, universeLeaves)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []UniverseLeafe
	for rows.Next() {
		var i UniverseLeafe
		if err := rows.Scan(
			&i.ID,
			&i.AssetGenesisID,
			&i.MintingPoint,
			&i.ScriptKeyBytes,
			&i.UniverseRootID,
			&i.LeafNodeKey,
			&i.LeafNodeNamespace,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const universeRoots = `-- name: UniverseRoots :many
SELECT universe_roots.asset_id, group_key, proof_type,
       mssmt_roots.root_hash AS root_hash, mssmt_nodes.sum AS root_sum,
       genesis_assets.asset_tag AS asset_name
FROM universe_roots
JOIN mssmt_roots
    ON universe_roots.namespace_root = mssmt_roots.namespace
JOIN mssmt_nodes
    ON mssmt_nodes.hash_key = mssmt_roots.root_hash 
       AND mssmt_nodes.namespace = mssmt_roots.namespace
JOIN genesis_assets
    ON genesis_assets.asset_id = universe_roots.asset_id
ORDER BY 
    CASE WHEN $1 = 0 THEN universe_roots.id END ASC,
    CASE WHEN $1 = 1 THEN universe_roots.id END DESC
LIMIT $3 OFFSET $2
`

type UniverseRootsParams struct {
	SortDirection interface{}
	NumOffset     int32
	NumLimit      int32
}

type UniverseRootsRow struct {
	AssetID   []byte
	GroupKey  []byte
	ProofType string
	RootHash  []byte
	RootSum   int64
	AssetName string
}

func (q *Queries) UniverseRoots(ctx context.Context, arg UniverseRootsParams) ([]UniverseRootsRow, error) {
	rows, err := q.db.QueryContext(ctx, universeRoots, arg.SortDirection, arg.NumOffset, arg.NumLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []UniverseRootsRow
	for rows.Next() {
		var i UniverseRootsRow
		if err := rows.Scan(
			&i.AssetID,
			&i.GroupKey,
			&i.ProofType,
			&i.RootHash,
			&i.RootSum,
			&i.AssetName,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const upsertFederationGlobalSyncConfig = `-- name: UpsertFederationGlobalSyncConfig :exec
INSERT INTO federation_global_sync_config (
    proof_type, allow_sync_insert, allow_sync_export
)
VALUES ($1, $2, $3)
ON CONFLICT(proof_type)
    DO UPDATE SET
    allow_sync_insert = $2,
    allow_sync_export = $3
`

type UpsertFederationGlobalSyncConfigParams struct {
	ProofType       string
	AllowSyncInsert bool
	AllowSyncExport bool
}

func (q *Queries) UpsertFederationGlobalSyncConfig(ctx context.Context, arg UpsertFederationGlobalSyncConfigParams) error {
	_, err := q.db.ExecContext(ctx, upsertFederationGlobalSyncConfig, arg.ProofType, arg.AllowSyncInsert, arg.AllowSyncExport)
	return err
}

const upsertFederationProofSyncLog = `-- name: UpsertFederationProofSyncLog :one
INSERT INTO federation_proof_sync_log AS log (
    status, timestamp, sync_direction, proof_leaf_id, universe_root_id,
    servers_id
) VALUES (
    $1, $2, $3,
    (
        -- Select the leaf id from the universe_leaves table.
        SELECT id
        FROM universe_leaves
        WHERE leaf_node_namespace = $4
            AND minting_point = $5
            AND script_key_bytes = $6
        LIMIT 1
    ),
    (
        -- Select the universe root id from the universe_roots table.
        SELECT id
        FROM universe_roots
        WHERE namespace_root = $7
        LIMIT 1
    ),
    (
        -- Select the server id from the universe_servers table.
        SELECT id
        FROM universe_servers
        WHERE server_host = $8
        LIMIT 1
    )
) ON CONFLICT (sync_direction, proof_leaf_id, universe_root_id, servers_id)
DO UPDATE SET
    status = EXCLUDED.status,
    timestamp = EXCLUDED.timestamp,
    -- Increment the attempt counter.
    attempt_counter = CASE
       WHEN $9 = TRUE THEN log.attempt_counter + 1
       ELSE log.attempt_counter
    END
RETURNING id
`

type UpsertFederationProofSyncLogParams struct {
	Status                 string
	Timestamp              time.Time
	SyncDirection          string
	LeafNamespace          string
	LeafMintingPointBytes  []byte
	LeafScriptKeyBytes     []byte
	UniverseIDNamespace    string
	ServerHost             string
	BumpSyncAttemptCounter interface{}
}

func (q *Queries) UpsertFederationProofSyncLog(ctx context.Context, arg UpsertFederationProofSyncLogParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, upsertFederationProofSyncLog,
		arg.Status,
		arg.Timestamp,
		arg.SyncDirection,
		arg.LeafNamespace,
		arg.LeafMintingPointBytes,
		arg.LeafScriptKeyBytes,
		arg.UniverseIDNamespace,
		arg.ServerHost,
		arg.BumpSyncAttemptCounter,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const upsertFederationUniSyncConfig = `-- name: UpsertFederationUniSyncConfig :exec
INSERT INTO federation_uni_sync_config  (
    namespace, asset_id, group_key, proof_type, allow_sync_insert, allow_sync_export
)
VALUES(
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT(namespace)
    DO UPDATE SET
    allow_sync_insert = $5,
    allow_sync_export = $6
`

type UpsertFederationUniSyncConfigParams struct {
	Namespace       string
	AssetID         []byte
	GroupKey        []byte
	ProofType       string
	AllowSyncInsert bool
	AllowSyncExport bool
}

func (q *Queries) UpsertFederationUniSyncConfig(ctx context.Context, arg UpsertFederationUniSyncConfigParams) error {
	_, err := q.db.ExecContext(ctx, upsertFederationUniSyncConfig,
		arg.Namespace,
		arg.AssetID,
		arg.GroupKey,
		arg.ProofType,
		arg.AllowSyncInsert,
		arg.AllowSyncExport,
	)
	return err
}

const upsertMultiverseLeaf = `-- name: UpsertMultiverseLeaf :one
INSERT INTO multiverse_leaves (
    multiverse_root_id, asset_id, group_key, leaf_node_key, leaf_node_namespace
) VALUES (
    $1, $2, $3, $4,
    $5
)
ON CONFLICT (leaf_node_key, leaf_node_namespace)
    -- This is a no-op to allow returning the ID.
    DO UPDATE SET leaf_node_key = EXCLUDED.leaf_node_key,
                  leaf_node_namespace = EXCLUDED.leaf_node_namespace
RETURNING id
`

type UpsertMultiverseLeafParams struct {
	MultiverseRootID  int64
	AssetID           []byte
	GroupKey          []byte
	LeafNodeKey       []byte
	LeafNodeNamespace string
}

func (q *Queries) UpsertMultiverseLeaf(ctx context.Context, arg UpsertMultiverseLeafParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, upsertMultiverseLeaf,
		arg.MultiverseRootID,
		arg.AssetID,
		arg.GroupKey,
		arg.LeafNodeKey,
		arg.LeafNodeNamespace,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const upsertMultiverseRoot = `-- name: UpsertMultiverseRoot :one
INSERT INTO multiverse_roots (namespace_root, proof_type)
VALUES ($1, $2)
ON CONFLICT (namespace_root)
    -- This is a no-op to allow returning the ID.
    DO UPDATE SET namespace_root = EXCLUDED.namespace_root
RETURNING id
`

type UpsertMultiverseRootParams struct {
	NamespaceRoot string
	ProofType     string
}

func (q *Queries) UpsertMultiverseRoot(ctx context.Context, arg UpsertMultiverseRootParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, upsertMultiverseRoot, arg.NamespaceRoot, arg.ProofType)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const upsertUniverseLeaf = `-- name: UpsertUniverseLeaf :exec
INSERT INTO universe_leaves (
    asset_genesis_id, script_key_bytes, universe_root_id, leaf_node_key, 
    leaf_node_namespace, minting_point
) VALUES (
    $1, $2, $3, $4,
    $5, $6
) ON CONFLICT (minting_point, script_key_bytes)
    -- This is a NOP, minting_point and script_key_bytes are the unique fields
    -- that caused the conflict.
    DO UPDATE SET minting_point = EXCLUDED.minting_point,
                  script_key_bytes = EXCLUDED.script_key_bytes
`

type UpsertUniverseLeafParams struct {
	AssetGenesisID    int64
	ScriptKeyBytes    []byte
	UniverseRootID    int64
	LeafNodeKey       []byte
	LeafNodeNamespace string
	MintingPoint      []byte
}

func (q *Queries) UpsertUniverseLeaf(ctx context.Context, arg UpsertUniverseLeafParams) error {
	_, err := q.db.ExecContext(ctx, upsertUniverseLeaf,
		arg.AssetGenesisID,
		arg.ScriptKeyBytes,
		arg.UniverseRootID,
		arg.LeafNodeKey,
		arg.LeafNodeNamespace,
		arg.MintingPoint,
	)
	return err
}

const upsertUniverseRoot = `-- name: UpsertUniverseRoot :one
INSERT INTO universe_roots (
    namespace_root, asset_id, group_key, proof_type
) VALUES (
    $1, $2, $3, $4
) ON CONFLICT (namespace_root)
    -- This is a NOP, namespace_root is the unique field that caused the
    -- conflict.
    DO UPDATE SET namespace_root = EXCLUDED.namespace_root
RETURNING id
`

type UpsertUniverseRootParams struct {
	NamespaceRoot string
	AssetID       []byte
	GroupKey      []byte
	ProofType     string
}

func (q *Queries) UpsertUniverseRoot(ctx context.Context, arg UpsertUniverseRootParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, upsertUniverseRoot,
		arg.NamespaceRoot,
		arg.AssetID,
		arg.GroupKey,
		arg.ProofType,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}
