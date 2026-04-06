package integrationtests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheClear(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	// Go up one directory since we're in integrationtests/
	workspaceDir := filepath.Join(currentDir, "..")

	var (
		buildDir   = filepath.Join(workspaceDir, "builds")
		binaryPath = filepath.Join(buildDir, "gobuildcache")
		testsDir   = filepath.Join(workspaceDir, "faketests")
		cacheDir   = filepath.Join(workspaceDir, "test-cache-clear")
	)

	// Clean up test cache directory at the end
	defer os.RemoveAll(cacheDir)

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

	// Clear Go's local cache to ensure clean state
	t.Log("Step 2: Clearing Go's local cache...")
	cleanCmd := exec.Command("go", "clean", "-cache")
	cleanCmd.Dir = workspaceDir
	if err := cleanCmd.Run(); err != nil {
		t.Logf("Warning: Failed to clean Go cache: %v", err)
	}

	t.Log("Step 3: Running tests to populate cache...")
	firstRunCmd := exec.Command("go", "test", "-v", testsDir)
	firstRunCmd.Dir = workspaceDir
	firstRunCmd.Env = append(os.Environ(),
		"GOCACHEPROG="+binaryPath,
		"GOBUILDCACHE_BACKEND_TYPE=disk",
		"GOBUILDCACHE_DEBUG=false",
		"GOBUILDCACHE_CACHE_DIR="+cacheDir)

	var firstRunOutput bytes.Buffer
	firstRunCmd.Stdout = &firstRunOutput
	firstRunCmd.Stderr = &firstRunOutput

	if err := firstRunCmd.Run(); err != nil {
		t.Fatalf("Tests failed on first run: %v\nOutput:\n%s", err, firstRunOutput.String())
	}
	t.Log("✓ Tests passed, cache populated")

	// Verify cache directory has files
	t.Log("Step 4: Verifying cache has entries...")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("Failed to read cache directory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("Cache directory is empty, expected cache entries")
	}
	t.Logf("✓ Cache has %d entries", len(entries))

	// Run tests again to verify they're cached
	t.Log("Step 5: Running tests again to verify caching works...")
	secondRunCmd := exec.Command("go", "test", "-v", testsDir)
	secondRunCmd.Dir = workspaceDir
	secondRunCmd.Env = append(os.Environ(),
		"GOCACHEPROG="+binaryPath,
		"GOBUILDCACHE_BACKEND_TYPE=disk",
		"GOBUILDCACHE_DEBUG=false",
		"GOBUILDCACHE_CACHE_DIR="+cacheDir)

	var secondRunOutput bytes.Buffer
	secondRunCmd.Stdout = &secondRunOutput
	secondRunCmd.Stderr = &secondRunOutput

	if err := secondRunCmd.Run(); err != nil {
		t.Fatalf("Tests failed on second run: %v\nOutput:\n%s", err, secondRunOutput.String())
	}

	if !strings.Contains(secondRunOutput.String(), "(cached)") {
		t.Fatal("Second run should be cached, but '(cached)' not found in output")
	}
	t.Log("✓ Tests were served from cache")

	// Now clear the cache
	t.Log("Step 6: Clearing the cache...")
	clearCmd := exec.Command(binaryPath, "clear", "-backend=disk", "-cache-dir="+cacheDir)
	clearCmd.Dir = workspaceDir
	clearOutput, err := clearCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to clear cache: %v\nOutput: %s", err, clearOutput)
	}
	t.Logf("✓ Cache cleared: %s", strings.TrimSpace(string(clearOutput)))

	// Verify cache directory is now empty or has no cache files
	t.Log("Step 7: Verifying cache is empty...")
	entries, err = os.ReadDir(cacheDir)
	if err != nil {
		// Directory might not exist anymore, which is fine
		if !os.IsNotExist(err) {
			t.Fatalf("Failed to read cache directory: %v", err)
		}
		t.Log("✓ Cache directory removed")
	} else {
		// Count actual cache files (not hidden temp files)
		cacheFiles := 0
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasPrefix(name, ".") {
				cacheFiles++
			}
		}
		if cacheFiles > 0 {
			t.Fatalf("Cache directory still has %d cache files after clear, expected 0", cacheFiles)
		}
		t.Log("✓ Cache directory is empty")
	}

	// Clear Go's local cache again to ensure third run is not cached by Go
	t.Log("Step 8: Clearing Go's local cache again...")
	cleanCmd = exec.Command("go", "clean", "-cache")
	cleanCmd.Dir = workspaceDir
	if err := cleanCmd.Run(); err != nil {
		t.Logf("Warning: Failed to clean Go cache: %v", err)
	}

	// Run tests again and verify they're NOT cached
	t.Log("Step 9: Running tests again to verify cache was cleared...")
	thirdRunCmd := exec.Command("go", "test", "-v", testsDir)
	thirdRunCmd.Dir = workspaceDir
	thirdRunCmd.Env = append(os.Environ(),
		"GOCACHEPROG="+binaryPath,
		"GOBUILDCACHE_BACKEND_TYPE=disk",
		"GOBUILDCACHE_DEBUG=false",
		"GOBUILDCACHE_CACHE_DIR="+cacheDir)

	var thirdRunOutput bytes.Buffer
	thirdRunCmd.Stdout = &thirdRunOutput
	thirdRunCmd.Stderr = &thirdRunOutput

	if err := thirdRunCmd.Run(); err != nil {
		t.Fatalf("Tests failed on third run: %v\nOutput:\n%s", err, thirdRunOutput.String())
	}

	if strings.Contains(thirdRunOutput.String(), "(cached)") {
		t.Fatalf("Third run should NOT be cached after clear, but found '(cached)' in output.\nOutput:\n%s", thirdRunOutput.String())
	}
	t.Log("✓ Tests were NOT cached (clear worked!)")

	t.Log("=== Cache clear test passed! ===")
}
