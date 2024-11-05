package tapdb

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/lightninglabs/taproot-assets/tapdb/sqlc"
	"github.com/lightninglabs/taproot-assets/universe"
	"github.com/lightningnetwork/lnd/clock"
	"github.com/stretchr/testify/require"
)


type dbSizeStats struct {
	tableName     string
	rowCount     int64
	dataSize     int64  // Size without indices
	indexSize    int64  // Total size of all indices
	totalSize    int64  // dataSize + indexSize
}

// measureDBSize fetches the size of tables and indexes for PostgreSQL and SQLite databases.
func measureDBSize(t *testing.T, db *BaseDB, isPostgres bool) map[string]*dbSizeStats {
	stats := make(map[string]*dbSizeStats)
	var rows *sql.Rows
	var err error

	if isPostgres {
		// Fetch table list and their size details for PostgreSQL
		rows, err = db.Query(`
			SELECT tablename, pg_table_size(tablename::regclass) AS data_size,
			       pg_indexes_size(tablename::regclass) AS index_size
			FROM pg_catalog.pg_tables
			WHERE schemaname != 'pg_catalog' AND schemaname != 'information_schema'
		`)
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var tableName string
			var dataSize, indexSize int64
			err := rows.Scan(&tableName, &dataSize, &indexSize)
			require.NoError(t, err)

			stats[tableName] = &dbSizeStats{
				tableName: tableName,
				dataSize:  dataSize,
				indexSize: indexSize,
				totalSize: dataSize + indexSize,
			}
		}
	} else {
		// SQLite: Fetch tables, calculate data size, and index size from dbstat
		rows, err = db.Query(`
			SELECT DISTINCT tbl_name
			FROM sqlite_master
			WHERE type='table'
		`)
		require.NoError(t, err)
		defer rows.Close()

		tables := make([]string, 0)
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			require.NoError(t, err)
			tables = append(tables, name)
			stats[name] = &dbSizeStats{
				tableName: name,
			}
		}

		// Calculate sizes for each table using dbstat for SQLite
		for _, tableName := range tables {
			var pageCount, pageSize, indexSize int64

			err = db.QueryRow(`SELECT COUNT(*) FROM dbstat WHERE name = ?`, tableName).Scan(&pageCount)
			if err != nil {
				t.Logf("Skipping size stats for %s: %v", tableName, err)
				continue
			}
			err = db.QueryRow(`PRAGMA page_size`).Scan(&pageSize)
			require.NoError(t, err)

			err = db.QueryRow(`SELECT COALESCE(SUM(pgsize), 0) FROM dbstat WHERE name = ? AND pagetype = 'index'`, tableName).Scan(&indexSize)
			if err != nil {
				t.Logf("Skipping index size for %s: %v", tableName, err)
				continue
			}

			stats[tableName].dataSize = pageCount * pageSize
			stats[tableName].indexSize = indexSize
			stats[tableName].totalSize = stats[tableName].dataSize + stats[tableName].indexSize
		}
	}
	return stats
}

// prettyPrintSizeStats formats the size statistics nicely
func prettyPrintSizeStats(t *testing.T, stats map[string]*dbSizeStats) {
	var totalData, totalIndex int64

	t.Log("\n=== Database Size Analysis ===")
	t.Log("Table                   Rows      Data Size    Index Size   Index Overhead")
	t.Log("----------------------------------------------------------------------")

	var tables []string
	for table := range stats {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	for _, table := range tables {
		stat := stats[table]
		if stat.dataSize == 0 {
			t.Logf("%-20s %8d %12s %12s %14s",
				stat.tableName,
				stat.rowCount,
				"-",
				"-",
				"-",
			)
			continue
		}

		overhead := float64(stat.indexSize) / float64(stat.dataSize) * 100
		t.Logf("%-20s %8d %12s %12s %14.1f%%",
			stat.tableName,
			stat.rowCount,
			formatSize(stat.dataSize),
			formatSize(stat.indexSize),
			overhead,
		)
		
		totalData += stat.dataSize
		totalIndex += stat.indexSize
	}

	t.Log("----------------------------------------------------------------------")
	if totalData > 0 {
		totalOverhead := float64(totalIndex) / float64(totalData) * 100
		t.Logf("%-20s %8s %12s %12s %14.1f%%",
			"TOTAL", "-",
			formatSize(totalData),
			formatSize(totalIndex),
			totalOverhead,
		)
	}
}

// formatSize returns human-readable file sizes
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(bytes)/float64(div), "KMGTPE"[exp])
}

// TestUniverseIndexPerformance tests query performance with the specified indices.
func TestUniverseIndexPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping index performance test in short mode")
	}

	t.Parallel()

	const (
		numAssets         = 50
		numLeavesPerTree  = 10
		numEventsPerAsset = 25
		numQueries        = 50
		batchSize         = 5
	)

	type queryStats struct {
		name           string
		withoutIndices time.Duration
		withIndices    time.Duration
		queries        int
	}
	testResults := make(map[string]*queryStats)

	// Function to run performance tests with and without indices
	runTest := func(withIndices bool) {
		t.Run(fmt.Sprintf("indices=%v", withIndices), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			db := NewTestDB(t)

			// Drop or create indices based on the test scenario
			sqlDB := db.BaseDB
			if !withIndices {
				t.Log("Dropping indices...")
				_, err := sqlDB.Exec(`
					DROP INDEX IF EXISTS idx_universe_roots_asset_group_proof;
					DROP INDEX IF EXISTS idx_universe_roots_proof_type_issuance;
					DROP INDEX IF EXISTS idx_universe_events_type_counts;
					DROP INDEX IF EXISTS idx_universe_events_universe_root_id;
					DROP INDEX IF EXISTS idx_universe_events_sync;
					DROP INDEX IF EXISTS idx_asset_group_witnesses_gen_asset_id;
					DROP INDEX IF EXISTS idx_mssmt_roots_hash_namespace;
					DROP INDEX IF EXISTS idx_genesis_assets_asset_id;
					DROP INDEX IF EXISTS idx_genesis_assets_asset_tag;
					DROP INDEX IF EXISTS idx_genesis_assets_asset_type;
					DROP INDEX IF EXISTS idx_universe_leaves_universe_root_id;
					DROP INDEX IF EXISTS idx_universe_leaves_asset_genesis_id;
					DROP INDEX IF EXISTS idx_universe_leaves_leaf_node_key_namespace;
				`)
				require.NoError(t, err)
			} else {
				t.Log("Creating indices...")
				_, err := sqlDB.Exec(`
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
				`)
				require.NoError(t, err)
			}

			testClock := clock.NewTestClock(time.Now())
			stats, _ := newUniverseStatsWithDB(db.BaseDB, testClock)
			h := newUniStatsHarness(t, numAssets, db.BaseDB, stats)

			// Generate and populate test data
			t.Log("Creating test data...")
			for i := 0; i < numAssets; i++ {
				for j := 0; j < numLeavesPerTree; j += batchSize {
					end := j + batchSize
					if end > numLeavesPerTree {
						end = numLeavesPerTree
					}
					for k := j; k < end; k++ {
						_, err := insertRandLeaf(t, ctx, h.assetUniverses[i], nil)
						require.NoError(t, err)
					}
				}
				for j := 0; j < numEventsPerAsset; j += batchSize {
					end := j + batchSize
					if end > numEventsPerAsset {
						end = numEventsPerAsset
					}
					for k := j; k < end; k++ {
						h.logProofEventByIndex(i)
						h.logSyncEventByIndex(i)
					}
				}
				if (i+1)%10 == 0 {
					t.Logf("Processed %d/%d assets", i+1, numAssets)
				}
			}

			// Measure initial database size
			isPostgres := db.Backend() == sqlc.BackendTypePostgres
			initialSizes := measureDBSize(t, db.BaseDB, isPostgres)
			prettyPrintSizeStats(t, initialSizes)

			// Analyze tables if using indices
			if withIndices {
				t.Log("Analyzing tables...")
				_, err := sqlDB.Exec("ANALYZE;")
				require.NoError(t, err)
			}

			// Test query performance
			testQueries := []struct {
				name string
				fn   func() time.Duration
			}{
				{
					name: "universe root namespace",
					fn: func() time.Duration {
						readTx := NewBaseUniverseReadTx()
						start := time.Now()
						err := h.assetUniverses[0].db.ExecTx(ctx, &readTx, func(db BaseUniverseStore) error {
							_, err := db.FetchUniverseRoot(ctx, h.assetUniverses[0].id.String())
							return err
						})
						require.NoError(t, err)
						return time.Since(start)
					},
				},
				// Add more query performance tests as needed
			}

			for _, q := range testQueries {
				var totalTime time.Duration
				for i := 0; i < numQueries; i++ {
					queryTime := q.fn()
					totalTime += queryTime
				}

				avgTime := totalTime / time.Duration(numQueries)
				t.Logf("%s average query time: %v", q.name, avgTime)

				stat, ok := testResults[q.name]
				if !ok {
					stat = &queryStats{name: q.name}
					testResults[q.name] = stat
				}

				if withIndices {
					stat.withIndices = avgTime
				} else {
					stat.withoutIndices = avgTime
				}
				stat.queries = numQueries
			}
		})
	}

	// Execute tests without and with indices
	runTest(false)
	runTest(true)

	// Print performance comparison
	t.Log("\n=== Performance Comparison ===")
	for name, result := range testResults {
		improvement := float64(result.withoutIndices) / float64(result.withIndices)
		t.Logf("\nQuery: %s (%d runs each)", name, result.queries)
		t.Logf("  With indices:    %v", result.withIndices)
		t.Logf("  Without indices: %v", result.withoutIndices)
		t.Logf("  Improvement:     %.2fx", improvement)
	}
}

// TestUniverseQuerySyncStatsSorting checks that query results are sorted correctly
func TestUniversePerfQuerySyncStatsSorting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sorting performance test in short mode")
	}

	t.Parallel()

	const numAssets = 50
	type queryStats struct {
		name           string
		withoutIndices time.Duration
		withIndices    time.Duration
		queries        int
	}
	testResults := make(map[string]*queryStats)

	// Function to run performance tests with and without indices
	runTest := func(withIndices bool) {
		t.Run(fmt.Sprintf("indices=%v", withIndices), func(t *testing.T) {
			db := NewTestDB(t)
			
			// Drop or create indices based on the test scenario
			sqlDB := db.BaseDB
			if !withIndices {
				t.Log("Dropping indices...")
				_, err := sqlDB.Exec(`
					DROP INDEX IF EXISTS idx_genesis_assets_asset_tag;
					DROP INDEX IF EXISTS idx_genesis_assets_asset_type;
					DROP INDEX IF EXISTS idx_genesis_assets_asset_id;
					DROP INDEX IF EXISTS idx_universe_events_type_counts;
					DROP INDEX IF EXISTS idx_universe_events_universe_root_id;
					DROP INDEX IF EXISTS idx_universe_events_sync;
				`)
				require.NoError(t, err)
			} else {
				t.Log("Creating indices...")
				_, err := sqlDB.Exec(`
					CREATE INDEX IF NOT EXISTS idx_genesis_assets_asset_tag 
					ON genesis_assets (asset_tag);
					CREATE INDEX IF NOT EXISTS idx_genesis_assets_asset_type 
					ON genesis_assets (asset_type);
					CREATE INDEX IF NOT EXISTS idx_genesis_assets_asset_id 
					ON genesis_assets (asset_id);
					CREATE INDEX IF NOT EXISTS idx_universe_events_type_counts 
					ON universe_events (event_type, universe_root_id);
					CREATE INDEX IF NOT EXISTS idx_universe_events_universe_root_id 
					ON universe_events (universe_root_id);
					CREATE INDEX IF NOT EXISTS idx_universe_events_sync 
					ON universe_events (event_type);
				`)
				require.NoError(t, err)
			}

			testClock := clock.NewTestClock(time.Now())
			statsDB, _ := newUniverseStatsWithDB(db.BaseDB, testClock)
			sh := newUniStatsHarness(t, numAssets, db.BaseDB, statsDB)

			// Log proof and sync events for each asset
			for i := 0; i < numAssets; i++ {
				sh.logProofEventByIndex(i)
				sh.logProofEventByIndex(i)
				numSyncs := rand.Int() % 10
				for j := 0; j < numSyncs; j++ {
					sh.logSyncEventByIndex(i)
				}
			}

			// Run sorting tests for various fields and directions
			tests := []struct {
				name      string
				sortType  universe.SyncStatsSort
				direction universe.SortDirection
			}{
				{"sort by name ascending", universe.SortByAssetName, universe.SortAscending},
				{"sort by name descending", universe.SortByAssetName, universe.SortDescending},
				{"sort by total syncs ascending", universe.SortByTotalSyncs, universe.SortAscending},
				{"sort by total syncs descending", universe.SortByTotalSyncs, universe.SortDescending},
				{"sort by total proofs ascending", universe.SortByTotalProofs, universe.SortAscending},
				{"sort by total proofs descending", universe.SortByTotalProofs, universe.SortDescending},
			}

			for _, test := range tests {
				var totalTime time.Duration
				const numQueries = 10

				for i := 0; i < numQueries; i++ {
					start := time.Now()
					syncStats, err := statsDB.QuerySyncStats(context.Background(), 
						universe.SyncStatsQuery{
							SortBy:        test.sortType,
							SortDirection: test.direction,
						})
					require.NoError(t, err)
					queryTime := time.Since(start)
					totalTime += queryTime

					// Verify sorting is correct
					require.True(t, sort.SliceIsSorted(
						syncStats.SyncStats,
						isSortedWithDirection(
							syncStats.SyncStats, test.sortType, 
							test.direction,
						),
					))
				}

				avgTime := totalTime / time.Duration(numQueries)
				t.Logf("%s average query time: %v", test.name, avgTime)

				stat, ok := testResults[test.name]
				if !ok {
					stat = &queryStats{name: test.name}
					testResults[test.name] = stat
				}

				if withIndices {
					stat.withIndices = avgTime
				} else {
					stat.withoutIndices = avgTime
				}
				stat.queries = numQueries
			}
		})
	}

	// Execute tests without and with indices
	runTest(false)
	runTest(true)

	// Print performance comparison
	t.Log("\n=== Performance Comparison ===")
	for name, result := range testResults {
		improvement := float64(result.withoutIndices) / float64(result.withIndices)
		t.Logf("\nQuery: %s (%d runs each)", name, result.queries)
		t.Logf("  With indices:    %v", result.withIndices)
		t.Logf("  Without indices: %v", result.withoutIndices)
		t.Logf("  Improvement:     %.2fx", improvement)
	}
}

// Helper for sorting checks
func isSortedWithDirection(s []universe.AssetSyncSnapshot, sortType universe.SyncStatsSort, 
	direction universe.SortDirection) func(i, j int) bool {
	
	asc := direction == universe.SortAscending

	return func(i, j int) bool {
		switch sortType {
		case universe.SortByAssetName:
			if asc {
				return s[i].AssetName < s[j].AssetName
			}
			return s[i].AssetName > s[j].AssetName

		case universe.SortByTotalSyncs:
			if asc {
				return s[i].TotalSyncs < s[j].TotalSyncs
			}
			return s[i].TotalSyncs > s[j].TotalSyncs

		case universe.SortByTotalProofs:
			if asc {
				return s[i].TotalProofs < s[j].TotalProofs
			}
			return s[i].TotalProofs > s[j].TotalProofs

		default:
			return false
		}
	}
}

func TestUniversePerfInserts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping insert performance test in short mode")
	}

	t.Parallel()

	const (
		numAssets = 50
		numLeavesPerTree = 10
		numEventsPerAsset = 25
	)

	var batchSizes = []int{1, 5, 10}

	type insertStats struct {
		name           string
		withoutIndices time.Duration
		withIndices    time.Duration
		batchSize      int
		totalInserts   int
	}
	testResults := make(map[string]*insertStats)

	// Function to run performance tests with and without indices
	runTest := func(withIndices bool) {
		t.Run(fmt.Sprintf("indices=%v", withIndices), func(t *testing.T) {
			ctx := context.Background()
			db := NewTestDB(t)
			
			// Drop or create indices based on test scenario
			sqlDB := db.BaseDB
			if !withIndices {
				t.Log("Dropping indices...")
				_, err := sqlDB.Exec(`
					DROP INDEX IF EXISTS idx_universe_roots_asset_group_proof;
					DROP INDEX IF EXISTS idx_universe_events_type_counts;
					DROP INDEX IF EXISTS idx_universe_events_universe_root_id;
					DROP INDEX IF EXISTS idx_universe_events_sync;
					DROP INDEX IF EXISTS idx_universe_leaves_universe_root_id;
					DROP INDEX IF EXISTS idx_universe_leaves_asset_genesis_id;
				`)
				require.NoError(t, err)
			} else {
				t.Log("Creating indices...")
				_, err := sqlDB.Exec(`
					CREATE INDEX IF NOT EXISTS idx_universe_roots_asset_group_proof 
					ON universe_roots (asset_id, group_key, proof_type);
					CREATE INDEX IF NOT EXISTS idx_universe_events_type_counts 
					ON universe_events (event_type, universe_root_id);
					CREATE INDEX IF NOT EXISTS idx_universe_events_universe_root_id 
					ON universe_events (universe_root_id);
					CREATE INDEX IF NOT EXISTS idx_universe_events_sync 
					ON universe_events (event_type);
					CREATE INDEX IF NOT EXISTS idx_universe_leaves_universe_root_id 
					ON universe_leaves (universe_root_id);
					CREATE INDEX IF NOT EXISTS idx_universe_leaves_asset_genesis_id 
					ON universe_leaves (asset_genesis_id);
				`)
				require.NoError(t, err)
			}

			testClock := clock.NewTestClock(time.Now())
			stats, _ := newUniverseStatsWithDB(db.BaseDB, testClock)
			h := newUniStatsHarness(t, numAssets, db.BaseDB, stats)

			// Test different batch sizes
			for _, batchSize := range batchSizes {
				var totalTime time.Duration
				totalInserts := 0

				// Insert leaves
				start := time.Now()
				for i := 0; i < numAssets; i++ {
					for j := 0; j < numLeavesPerTree; j += batchSize {
						end := j + batchSize
						if end > numLeavesPerTree {
							end = numLeavesPerTree
						}
						for k := j; k < end; k++ {
							_, err := insertRandLeaf(t, ctx, h.assetUniverses[i], nil)
							require.NoError(t, err)
							totalInserts++
						}
					}
				}
				leafTime := time.Since(start)

				// Insert events
				start = time.Now()
				for i := 0; i < numAssets; i++ {
					for j := 0; j < numEventsPerAsset; j += batchSize {
						end := j + batchSize
						if end > numEventsPerAsset {
							end = numEventsPerAsset
						}
						for k := j; k < end; k++ {
							h.logProofEventByIndex(i)
							h.logSyncEventByIndex(i)
							totalInserts += 2
						}
					}
				}
				eventTime := time.Since(start)
				totalTime = leafTime + eventTime

				name := fmt.Sprintf("batch_size_%d", batchSize)
				stat, ok := testResults[name]
				if !ok {
					stat = &insertStats{
						name:         name,
						batchSize:    batchSize,
						totalInserts: totalInserts,
					}
					testResults[name] = stat
				}

				if withIndices {
					stat.withIndices = totalTime
				} else {
					stat.withoutIndices = totalTime
				}
			}
		})
	}

	// Execute tests without and with indices
	runTest(false)
	runTest(true)

	// Print performance comparison
	t.Log("\n=== Insert Performance Comparison ===")
	for _, result := range testResults {
		improvement := float64(result.withoutIndices) / float64(result.withIndices)
		opsPerSecWithIndices := float64(result.totalInserts) / result.withIndices.Seconds()
		opsPerSecWithoutIndices := float64(result.totalInserts) / result.withoutIndices.Seconds()

		t.Logf("\nBatch size: %d (%d total inserts)", 
			result.batchSize, result.totalInserts)
		t.Logf("  With indices:    %v (%.0f ops/sec)", 
			result.withIndices, opsPerSecWithIndices)
		t.Logf("  Without indices: %v (%.0f ops/sec)", 
			result.withoutIndices, opsPerSecWithoutIndices)
		t.Logf("  Ratio:          %.2fx", improvement)
	}
}