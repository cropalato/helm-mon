package repository

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

func setupTestEnvironment(t *testing.T) (*Manager, string, func()) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "helm-repo-test-*")
	require.NoError(t, err)

	// Create test settings
	settings := cli.New()
	settings.RepositoryConfig = filepath.Join(tmpDir, "repositories.yaml")
	settings.RepositoryCache = filepath.Join(tmpDir, "cache")

	// Create cache directory
	err = os.MkdirAll(settings.RepositoryCache, 0755)
	require.NoError(t, err)

	// Create test logger
	logger := zaptest.NewLogger(t)

	// Create repository manager
	manager := NewManager(settings, logger)

	// Return cleanup function
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return manager, tmpDir, cleanup
}

func createTestRepositoryFile(t *testing.T, path string, repos []*repo.Entry) {
	file := &repo.File{
		APIVersion:   "v1",
		Generated:    time.Now(),
		Repositories: repos,
	}

	err := file.WriteFile(path, 0644)
	require.NoError(t, err)
}

func TestLoadRepositories(t *testing.T) {
	manager, tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	tests := []struct {
		name           string
		repositories   []*repo.Entry
		expectError    bool
		expectedCount  int
		setupIndexFile bool
	}{
		{
			name: "Successfully load repositories",
			repositories: []*repo.Entry{
				{Name: "stable", URL: "https://charts.helm.sh/stable"},
				{Name: "bitnami", URL: "https://charts.bitnami.com/bitnami"},
			},
			expectError:    false,
			expectedCount:  2,
			setupIndexFile: true,
		},
		{
			name:           "Empty repository list",
			repositories:   []*repo.Entry{},
			expectError:    false,
			expectedCount:  0,
			setupIndexFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create repository configuration
			createTestRepositoryFile(t, manager.settings.RepositoryConfig, tt.repositories)

			if tt.setupIndexFile {
				// Create dummy index files
				for _, repo := range tt.repositories {
					indexFile := repo.IndexFile()
					indexFile.Generated = time.Now()
					cacheFile := filepath.Join(manager.settings.RepositoryCache, "index-"+repo.Name+".yaml")
					err := indexFile.WriteFile(cacheFile, 0644)
					require.NoError(t, err)
				}
			}

			// Test repository loading
			repos, err := manager.LoadRepositories()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, repos, tt.expectedCount)
			}
		})
	}
}

func TestGetRepositoryInfo(t *testing.T) {
	manager, tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Setup test repositories
	testRepos := []*repo.Entry{
		{Name: "stable", URL: "https://charts.helm.sh/stable"},
		{Name: "bitnami", URL: "https://charts.bitnami.com/bitnami"},
	}

	createTestRepositoryFile(t, manager.settings.RepositoryConfig, testRepos)

	// Create test index files
	for _, repo := range testRepos {
		indexFile := repo.IndexFile()
		indexFile.Generated = time.Now()
		// Add some test charts
		indexFile.Entries = map[string]repo.ChartVersions{
			"test-chart": {},
		}
		cacheFile := filepath.Join(manager.settings.RepositoryCache, "index-"+repo.Name+".yaml")
		err := indexFile.WriteFile(cacheFile, 0644)
		require.NoError(t, err)
	}

	// Test getting repository info
	info, err := manager.GetRepositoryInfo()
	require.NoError(t, err)
	assert.Len(t, info, len(testRepos))

	// Verify repository information
	for i, repo := range testRepos {
		assert.Equal(t, repo.Name, info[i].Name)
		assert.Equal(t, repo.URL, info[i].URL)
		assert.True(t, info[i].HasIndexFile)
		assert.Equal(t, 1, info[i].ChartCount)
	}
}

func TestSyncRepositories(t *testing.T) {
	manager, tmpDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Setup test repositories
	testRepos := []*repo.Entry{
		{Name: "stable", URL: "https://charts.helm.sh/stable"},
	}

	createTestRepositoryFile(t, manager.settings.RepositoryConfig, testRepos)

	// Test repository sync
	result, err := manager.SyncRepositories()

	// Note: In a real test environment, you would mock the HTTP requests
	// Here we expect the sync to fail because we're not mocking external requests
	assert.NoError(t, err)
	assert.Len(t, result.Failed, 1)
}
