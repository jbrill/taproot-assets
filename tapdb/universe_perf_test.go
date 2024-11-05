package tapdb

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

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

// measureDBSize gets table and index sizes from SQLite
func measureDBSize(t *testing.T, db *BaseDB) map[string]*dbSizeStats {
	stats := make(map[string]*dbSizeStats)

	// First get list of all tables
	rows, err := db.Query(`
		SELECT DISTINCT
			tbl_name,
			type
		FROM sqlite_master
		WHERE type='table'
	`)
	require.NoError(t, err)
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var name, tblType string
		err := rows.Scan(&name, &tblType)
		require.NoError(t, err)
		tables = append(tables, name)

		stats[name] = &dbSizeStats{
			tableName: name,
		}
	}

	// For each table, get its stats
	for _, tableName := range tables {
		// Get row count using COUNT(*)
		var rowCount int64
		err = db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM main.%q`, tableName)).Scan(&rowCount)
		if err != nil {
			t.Logf("Skipping row count for %s: %v", tableName, err)
			continue
		}
		stats[tableName].rowCount = rowCount

		// Get table size
		var pageCount int64
		err = db.QueryRow(`
			SELECT COUNT(*) 
			FROM dbstat 
			WHERE name = ?
		`, tableName).Scan(&pageCount)
		if err != nil {
			t.Logf("Skipping size stats for %s: %v", tableName, err)
			continue
		}

		// Get page size (constant for the database)
		var pageSize int64
		err = db.QueryRow(`PRAGMA page_size`).Scan(&pageSize)
		require.NoError(t, err)

		stats[tableName].dataSize = pageCount * pageSize
	}

	// Get list of indices and their sizes
	rows, err = db.Query(`
		SELECT 
			m.tbl_name as table_name,
			m.name as index_name,
			(SELECT COUNT(*) FROM dbstat WHERE name = m.name) as page_count,
			(SELECT page_size FROM pragma_page_size) as page_size
		FROM sqlite_master m
		WHERE m.type = 'index'
	`)
	require.NoError(t, err)
	defer rows.Close()

	for rows.Next() {
		var (
			tableName string
			indexName string
			pageCount int64
			pageSize  int64
		)
		err := rows.Scan(&tableName, &indexName, &pageCount, &pageSize)
		if err != nil {
			t.Logf("Skipping index stat: %v", err)
			continue
		}

		if stat, ok := stats[tableName]; ok {
			indexSize := pageCount * pageSize
			stat.indexSize += indexSize
			stat.totalSize = stat.dataSize + stat.indexSize
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

// TestUniverseIndexPerformance tests that our new indices improve query
// performance by comparing performance with and without indices.
func TestUniverseIndexPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping index performance test in short mode")
	}

	t.Parallel()

	const (
		numAssets         = 25
		numLeavesPerTree  = 10
		numEventsPerAsset = 15
		numQueries       = 10
		batchSize        = 5
	)

	type queryStats struct {
		name           string
		withoutIndices time.Duration
		withIndices    time.Duration
		queries        int
	}
	testResults := make(map[string]*queryStats)

	// Run without indices first, then with indices
	runTest := func(withIndices bool) {
		t.Run(fmt.Sprintf("indices=%v", withIndices), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(
				context.Background(), time.Minute,
			)
			defer cancel()

			db := NewTestDB(t)

			// Drop indices only if we're testing without them
			if !withIndices {
				t.Log("Dropping indices...")
				sqlDB := db.BaseDB
				_, err := sqlDB.Exec(`
					DROP INDEX IF EXISTS idx_universe_roots_namespace;
					DROP INDEX IF EXISTS idx_universe_roots_issuance;
					DROP INDEX IF EXISTS idx_universe_leaves_lookup;
					DROP INDEX IF EXISTS idx_universe_leaves_sort;
					DROP INDEX IF EXISTS idx_mssmt_nodes_namespace;
					DROP INDEX IF EXISTS idx_mssmt_nodes_key_lookup;
					DROP INDEX IF EXISTS idx_universe_events_stats;
					DROP INDEX IF EXISTS idx_universe_events_root_type;
					DROP INDEX IF EXISTS idx_federation_sync_composite;
					DROP INDEX IF EXISTS idx_multiverse_leaves_composite;
				`)
				require.NoError(t, err)
			}

			testClock := clock.NewTestClock(time.Now())
			stats, _ := newUniverseStatsWithDB(db.BaseDB, testClock)
			h := newUniStatsHarness(t, numAssets, db.BaseDB, stats)

			t.Log("Creating test data...")
			dataStart := time.Now()

			// Create test data in batches
			for i := 0; i < numAssets; i++ {
				// Create leaves in batches
				for j := 0; j < numLeavesPerTree; j += batchSize {
					end := j + batchSize
					if end > numLeavesPerTree {
						end = numLeavesPerTree
					}
					for k := j; k < end; k++ {
						_, err := insertRandLeaf(
							t, ctx, h.assetUniverses[i], nil,
						)
						require.NoError(t, err)
					}
				}

				// Create events in batches
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

			t.Logf("Test data creation took: %v", time.Since(dataStart))

			// Measure size after data creation
			t.Log("Measuring initial database size...")
			initialSizes := measureDBSize(t, db.BaseDB)
			prettyPrintSizeStats(t, initialSizes)

			if withIndices {
				t.Log("Analyzing tables...")
				sqlDB := db.BaseDB
				_, err := sqlDB.Exec("ANALYZE;")
				require.NoError(t, err)
			}

			testQueries := []struct {
				name string
				fn   func() time.Duration
				}{
				{
					name: "universe root namespace",
					fn: func() time.Duration {
						readTx := NewBaseUniverseReadTx()
						start := time.Now()
						err := h.assetUniverses[0].db.ExecTx(
							ctx, &readTx,
							func(db BaseUniverseStore) error {
								_, err := db.FetchUniverseRoot(
									ctx,
									h.assetUniverses[0].id.String(),
								)
								return err
							})
						require.NoError(t, err)
						return time.Since(start)
					},
				},
				{
					name: "universe events by type",
					fn: func() time.Duration {
						readTx := NewUniverseStatsReadTx()
						start := time.Now()
						err := stats.db.ExecTx(ctx, &readTx,
							func(db UniverseStatsStore) error {
								_, err := db.QueryAssetStatsPerDaySqlite(
									ctx, AssetStatsPerDayQuery{
										StartTime: testClock.Now().Add(
											-24 * time.Hour,
										).Unix(),
										EndTime: testClock.Now().Unix(),
									},
								)
								return err
							})
						require.NoError(t, err)
						return time.Since(start)
					},
				},
				{
					name: "universe leaves namespace",
					fn: func() time.Duration {
						readTx := NewBaseUniverseReadTx()
						start := time.Now()
						err := h.assetUniverses[0].db.ExecTx(
							ctx, &readTx,
							func(db BaseUniverseStore) error {
								_, err := db.QueryUniverseLeaves(
									ctx, UniverseLeafQuery{
										Namespace: h.assetUniverses[0].id.String(),
									},
								)
								return err
							})
						require.NoError(t, err)
						return time.Since(start)
					},
				},
			}

			// Run each query type multiple times
			for _, q := range testQueries {
				var totalTime time.Duration
				for i := 0; i < numQueries; i++ {
					queryTime := q.fn()
					totalTime += queryTime
				}

				avgTime := totalTime / time.Duration(numQueries)
				t.Logf("%s average query time: %v", q.name, avgTime)

				// Store result
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

	// Run tests in sequence - without indices first
	runTest(false)
	runTest(true)

	// Print final comparison
	t.Log("\n=== Performance Comparison ===")
	
	var testNames []string
	for name := range testResults {
		testNames = append(testNames, name)
	}
	sort.Strings(testNames)

	for _, name := range testNames {
		result := testResults[name]
		improvement := float64(result.withoutIndices) / float64(result.withIndices)
		t.Logf("\nQuery: %s (%d runs each)", name, result.queries)
		t.Logf("  With indices:    %v", result.withIndices)
		t.Logf("  Without indices: %v", result.withoutIndices)
		t.Logf("  Improvement:     %.2fx", improvement)
	}
}