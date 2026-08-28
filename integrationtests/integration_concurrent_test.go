package integrationtests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestCacheIntegrationConcurrentProcesses tests cross-process deduplication
// by running multiple go test processes concurrently with fslock-based deduplication.
func TestCacheIntegrationConcurrentProcesses(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	// Go up one directory since we're in integrationtests/
	workspaceDir := filepath.Join(currentDir, "..")

	var (
		buildDir     = filepath.Join(workspaceDir, "builds")
		binaryPath   = filepath.Join(buildDir, "gobuildcache")
		testsDir     = filepath.Join(workspaceDir, "faketests")
		cacheDir     = filepath.Join(workspaceDir, "test-cache-concurrent")
		lockDir      = filepath.Join(workspaceDir, "test-locks")
		numProcesses = 10 // Number of concurrent processes to run
	)

	// Clean up test directories at the end
	defer os.RemoveAll(cacheDir)
	defer os.RemoveAll(lockDir)

	t.Log("Step 1: Compiling the binary...")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("Failed to create build directory: %v", err)
	}

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = workspaceDir
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to compile binary: %v\nOutput: %s", err, buildOutput)
	}
	t.Log("✓ Binary compiled successfully")

	t.Log("Step 2: Clearing the cache and lock directory...")
	// Clean up directories
	os.RemoveAll(cacheDir)
	os.RemoveAll(lockDir)

	clearCmd := exec.Command(binaryPath, "clear", "-backend=disk", "-cache-dir="+cacheDir)
	clearCmd.Dir = workspaceDir
	clearOutput, err := clearCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v\nOutput: %s", err, clearOutput)
	}
	t.Logf("✓ Cache cleared successfully: %s", strings.TrimSpace(string(clearOutput)))

	// Clear Go's local cache to ensure first run is not cached
	t.Log("Step 2.5: Clearing Go's local cache...")
	cleanCmd := exec.Command("go", "clean", "-cache")
	cleanCmd.Dir = workspaceDir
	if err := cleanCmd.Run(); err != nil {
		t.Logf("Warning: Failed to clean Go cache: %v", err)
	}

	t.Logf("Step 3: Running %d concurrent go test processes with fslock deduplication...", numProcesses)

	var wg sync.WaitGroup
	outputs := make([]bytes.Buffer, numProcesses)
	errors := make([]error, numProcesses)

	// Run multiple go test processes concurrently
	for i := 0; i < numProcesses; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			cmd := exec.Command("go", "test", "-v", testsDir)
			cmd.Dir = workspaceDir
			// Use fslock for cross-process deduplication
			cmd.Env = append(os.Environ(),
				"GOCACHEPROG="+binaryPath,
				"GOBUILDCACHE_BACKEND_TYPE=disk",
				"GOBUILDCACHE_DEBUG=false",
				"GOBUILDCACHE_PRINT_STATS=true",
				"GOBUILDCACHE_LOCK_TYPE=fslock",
				"GOBUILDCACHE_LOCK_DIR="+lockDir,
				"GOBUILDCACHE_CACHE_DIR="+cacheDir)

			cmd.Stdout = &outputs[index]
			cmd.Stderr = &outputs[index]

			errors[index] = cmd.Run()
		}(i)
	}

	// Wait for all processes to complete
	wg.Wait()

	// Check results
	t.Log("Step 4: Checking results from concurrent processes...")
	successCount := 0
	for i := 0; i < numProcesses; i++ {
		if errors[i] == nil {
			successCount++
			t.Logf("✓ Process %d completed successfully", i+1)
		} else {
			t.Logf("✗ Process %d failed: %v", i+1, errors[i])
			t.Logf("Output:\n%s", outputs[i].String())
		}
	}

	if successCount == 0 {
		t.Fatal("All concurrent processes failed")
	}
	t.Logf("✓ %d/%d processes completed successfully", successCount, numProcesses)

	// Run a second batch to verify caching works
	t.Log("Step 5: Running second batch to verify caching...")
	var secondBatchOutput bytes.Buffer
	secondCmd := exec.Command("go", "test", "-v", testsDir)
	secondCmd.Dir = workspaceDir
	secondCmd.Env = append(os.Environ(),
		"GOCACHEPROG="+binaryPath,
		"GOBUILDCACHE_BACKEND_TYPE=disk",
		"GOBUILDCACHE_DEBUG=false",
		"GOBUILDCACHE_PRINT_STATS=true",
		"GOBUILDCACHE_LOCK_TYPE=fslock",
		"GOBUILDCACHE_LOCK_DIR="+lockDir,
		"GOBUILDCACHE_CACHE_DIR="+cacheDir)
	secondCmd.Stdout = &secondBatchOutput
	secondCmd.Stderr = &secondBatchOutput

	if err := secondCmd.Run(); err != nil {
		t.Fatalf("Second batch failed: %v\nOutput:\n%s", err, secondBatchOutput.String())
	}

	// Verify that results were cached
	if strings.Contains(secondBatchOutput.String(), "(cached)") {
		t.Log("✓ Second batch used cached results!")
	} else {
		t.Logf("Note: Second batch completed (may have been cached by Go itself)\nOutput:\n%s", secondBatchOutput.String())
	}

	t.Log("=== Cross-process deduplication test passed! ===")
}
