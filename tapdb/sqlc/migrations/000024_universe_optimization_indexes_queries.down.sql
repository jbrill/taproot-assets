-- Drop indices on universe_roots
DROP INDEX IF EXISTS idx_universe_roots_asset_group_proof;
DROP INDEX IF EXISTS idx_universe_roots_proof_type_issuance;

-- Drop indices on universe_events
DROP INDEX IF EXISTS idx_universe_events_type_counts;
DROP INDEX IF EXISTS idx_universe_events_universe_root_id;
DROP INDEX IF EXISTS idx_universe_events_sync;

-- Drop indices on tables underlying key_group_info_view
DROP INDEX IF EXISTS idx_asset_group_witnesses_gen_asset_id;

-- Drop indices on mssmt_roots
DROP INDEX IF EXISTS idx_mssmt_roots_hash_namespace;

-- Drop indices on genesis_assets
DROP INDEX IF EXISTS idx_genesis_assets_asset_id;
DROP INDEX IF EXISTS idx_genesis_assets_asset_tag;
DROP INDEX IF EXISTS idx_genesis_assets_asset_type;

-- Drop indices on universe_leaves
DROP INDEX IF EXISTS idx_universe_leaves_universe_root_id;
DROP INDEX IF EXISTS idx_universe_leaves_asset_genesis_id;
DROP INDEX IF EXISTS idx_universe_leaves_leaf_node_key_namespace;