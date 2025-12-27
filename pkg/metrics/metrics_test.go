package metrics

import (
	"testing"
	"time"
)

func TestLatencyTracker(t *testing.T) {
	tracker := NewLatencyTracker(0.01)

	// Record some sample latencies for different operations
	operations := []string{"get_overall", "put_overall", "get_backend", "put_backend"}

	for _, op := range operations {
		// Record a variety of latencies
		tracker.Record(op, 1*time.Millisecond)
		tracker.Record(op, 5*time.Millisecond)
		tracker.Record(op, 10*time.Millisecond)
		tracker.Record(op, 50*time.Millisecond)
		tracker.Record(op, 100*time.Millisecond)
	}

	// Test GetStats for each operation
	for _, op := range operations {
		stats, err := tracker.GetStats(op)
		if err != nil {
			t.Errorf("Failed to get stats for %s: %v", op, err)
			continue
		}

		if stats.Count != 5 {
			t.Errorf("Expected count 5 for %s, got %d", op, stats.Count)
		}

		if stats.Min < 0.9 || stats.Min > 1.1 {
			t.Errorf("Expected min ~1ms for %s, got %.2fms", op, stats.Min)
		}

		if stats.Max < 99 || stats.Max > 101 {
			t.Errorf("Expected max ~100ms for %s, got %.2fms", op, stats.Max)
		}

		// P50 should be around 10ms
		if stats.P50 < 5 || stats.P50 > 15 {
			t.Errorf("Expected p50 ~10ms for %s, got %.2fms", op, stats.P50)
		}

		// P99 should be reasonably high (we only have 5 samples, so it's approximate)
		if stats.P99 < 40 || stats.P99 > 110 {
			t.Errorf("Expected p99 between 40-110ms for %s, got %.2fms", op, stats.P99)
		}
	}

	// Test GetAllStats
	allStats := tracker.GetAllStats()
	if len(allStats) != len(operations) {
		t.Errorf("Expected %d operations in GetAllStats, got %d", len(operations), len(allStats))
	}

	// Test non-existent operation
	_, err := tracker.GetStats("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent operation, got nil")
	}
}

func TestLatencyTrackerRecordFunc(t *testing.T) {
	tracker := NewLatencyTracker(0.01)

	// Test RecordFunc
	err := tracker.RecordFunc("test_op", func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("RecordFunc returned error: %v", err)
	}

	stats, err := tracker.GetStats("test_op")
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Count != 1 {
		t.Errorf("Expected count 1, got %d", stats.Count)
	}

	// Should be at least 10ms
	if stats.Min < 9 {
		t.Errorf("Expected min >= 9ms, got %.2fms", stats.Min)
	}
}

func TestLatencyTrackerRecordFuncWithResult(t *testing.T) {
	tracker := NewLatencyTracker(0.01)

	// Test RecordFuncWithResult
	result, err := tracker.RecordFuncWithResult("test_op_result", func() (interface{}, error) {
		time.Sleep(5 * time.Millisecond)
		return "test_result", nil
	})

	if err != nil {
		t.Errorf("RecordFuncWithResult returned error: %v", err)
	}

	if result != "test_result" {
		t.Errorf("Expected result 'test_result', got %v", result)
	}

	stats, err := tracker.GetStats("test_op_result")
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.Count != 1 {
		t.Errorf("Expected count 1, got %d", stats.Count)
	}
}

func TestStatsString(t *testing.T) {
	stats := Stats{
		Operation: "test_op",
		Count:     100,
		Min:       1.5,
		P50:       10.2,
		P90:       50.7,
		P95:       75.3,
		P99:       99.1,
		Max:       120.5,
	}

	str := stats.String()
	expected := "  test_op (n=100): min=1.50ms p50=10.20ms p90=50.70ms p95=75.30ms p99=99.10ms max=120.50ms"
	if str != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, str)
	}

	// Test empty stats
	emptyStats := Stats{Operation: "empty_op"}
	emptyStr := emptyStats.String()
	expectedEmpty := "  empty_op: no data"
	if emptyStr != expectedEmpty {
		t.Errorf("Expected:\n%s\nGot:\n%s", expectedEmpty, emptyStr)
	}
}

func BenchmarkLatencyTrackerRecord(b *testing.B) {
	tracker := NewLatencyTracker(0.01)
	duration := 10 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Record("bench_op", duration)
	}
}

func BenchmarkLatencyTrackerGetStats(b *testing.B) {
	tracker := NewLatencyTracker(0.01)

	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		tracker.Record("bench_op", time.Duration(i)*time.Microsecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.GetStats("bench_op")
	}
}
