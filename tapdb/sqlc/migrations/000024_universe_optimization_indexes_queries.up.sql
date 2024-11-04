-- Core universe_roots indexes
CREATE INDEX IF NOT EXISTS idx_universe_roots_namespace 
ON universe_roots(namespace_root, id, asset_id, group_key, proof_type);

CREATE INDEX IF NOT EXISTS idx_universe_roots_issuance 
ON universe_roots(proof_type, group_key, id, asset_id, namespace_root);

-- Universe leaves optimization
CREATE INDEX IF NOT EXISTS idx_universe_leaves_lookup 
ON universe_leaves(
    leaf_node_namespace, minting_point, script_key_bytes,
    id, leaf_node_key, universe_root_id, asset_genesis_id
);

CREATE INDEX IF NOT EXISTS idx_universe_leaves_sort 
ON universe_leaves(leaf_node_namespace, id, minting_point, script_key_bytes);

-- MSSMT nodes optimization
CREATE INDEX IF NOT EXISTS idx_mssmt_nodes_namespace 
ON mssmt_nodes(namespace, hash_key, key, value, sum);

CREATE INDEX IF NOT EXISTS idx_mssmt_nodes_key_lookup 
ON mssmt_nodes(key, namespace, hash_key, value, sum);

-- Universe events optimization
CREATE INDEX IF NOT EXISTS idx_universe_events_stats 
ON universe_events(event_type, event_timestamp);

CREATE INDEX IF NOT EXISTS idx_universe_events_root_type 
ON universe_events(universe_root_id, event_type);

-- Federation sync log optimization
CREATE INDEX IF NOT EXISTS idx_federation_sync_composite 
ON federation_proof_sync_log(
    sync_direction, proof_leaf_id, universe_root_id, servers_id,
    id, status, timestamp, attempt_counter
);

-- Multiverse optimization
CREATE INDEX IF NOT EXISTS idx_multiverse_leaves_composite 
ON multiverse_leaves(
    multiverse_root_id, leaf_node_namespace,
    asset_id, group_key, leaf_node_key
);

-- Analyze existing tables
ANALYZE universe_roots;
ANALYZE universe_leaves;
ANALYZE mssmt_nodes;
ANALYZE universe_events;
ANALYZE federation_proof_sync_log;
ANALYZE multiverse_leaves;
ANALYZE multiverse_roots;