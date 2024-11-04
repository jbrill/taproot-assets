
-- Drop in reverse order of dependencies
DROP INDEX IF EXISTS idx_multiverse_leaves_composite;
DROP INDEX IF EXISTS idx_federation_sync_composite;
DROP INDEX IF EXISTS idx_universe_events_root_type;
DROP INDEX IF EXISTS idx_universe_events_stats;
DROP INDEX IF EXISTS idx_mssmt_nodes_key_lookup;
DROP INDEX IF EXISTS idx_mssmt_nodes_namespace;
DROP INDEX IF EXISTS idx_universe_leaves_sort;
DROP INDEX IF EXISTS idx_universe_leaves_lookup;
DROP INDEX IF EXISTS idx_universe_roots_issuance;
DROP INDEX IF EXISTS idx_universe_roots_namespace;

-- Update statistics
ANALYZE universe_roots;
ANALYZE universe_leaves;
ANALYZE mssmt_nodes;
ANALYZE universe_events;
ANALYZE federation_proof_sync_log;
ANALYZE multiverse_leaves;
ANALYZE multiverse_roots;