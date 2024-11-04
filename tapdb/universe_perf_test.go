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