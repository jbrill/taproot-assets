-- Composite index supporting joins and GROUP BY
CREATE INDEX IF NOT EXISTS idx_universe_roots_asset_group_proof 
ON universe_roots (asset_id, group_key, proof_type);

-- Partial index for proof_type = 'issuance'
CREATE INDEX IF NOT EXISTS idx_universe_roots_proof_type_issuance 
ON universe_roots (proof_type);

-- Composite index supporting event_type and universe_root_id
CREATE INDEX IF NOT EXISTS idx_universe_events_type_counts 
ON universe_events (event_type, universe_root_id);

-- Separate index on universe_root_id
CREATE INDEX IF NOT EXISTS idx_universe_events_universe_root_id 
ON universe_events (universe_root_id);

-- Partial index for event_type = 'SYNC'
CREATE INDEX IF NOT EXISTS idx_universe_events_sync 
ON universe_events (event_type);

-- Indices on tables underlying key_group_info_view
CREATE INDEX IF NOT EXISTS idx_asset_group_witnesses_gen_asset_id 
ON asset_group_witnesses (gen_asset_id);

-- Indices on mssmt_roots
CREATE INDEX IF NOT EXISTS idx_mssmt_roots_hash_namespace 
ON mssmt_roots (root_hash, namespace);

-- Indices on genesis_assets
CREATE INDEX IF NOT EXISTS idx_genesis_assets_asset_id 
ON genesis_assets (asset_id);
CREATE INDEX IF NOT EXISTS idx_genesis_assets_asset_tag 
ON genesis_assets (asset_tag);
CREATE INDEX IF NOT EXISTS idx_genesis_assets_asset_type 
ON genesis_assets (asset_type);

-- Indices on universe_leaves
CREATE INDEX IF NOT EXISTS idx_universe_leaves_universe_root_id 
ON universe_leaves (universe_root_id);
CREATE INDEX IF NOT EXISTS idx_universe_leaves_asset_genesis_id 
ON universe_leaves (asset_genesis_id);
CREATE INDEX IF NOT EXISTS idx_universe_leaves_leaf_node_key_namespace 
ON universe_leaves (leaf_node_key, leaf_node_namespace);