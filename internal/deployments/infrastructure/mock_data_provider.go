package infrastructure

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain/ports"
)

// MockDataProvider generates mock deployment data for testing
type MockDataProvider struct {
	repo ports.DeploymentRepository
	rand *rand.Rand
}

// NewMockDataProvider creates a new mock data provider
func NewMockDataProvider(repo ports.DeploymentRepository) *MockDataProvider {
	return &MockDataProvider{
		repo: repo,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// MockDataConfig contains configuration for generating mock data
type MockDataConfig struct {
	Count     int
	StartDate time.Time
	EndDate   time.Time
	Projects  []string
}

// DefaultMockDataConfig returns a default configuration
func DefaultMockDataConfig() MockDataConfig {
	now := time.Now()
	return MockDataConfig{
		Count:     10,
		StartDate: now.AddDate(0, -1, 0), // 1 month ago
		EndDate:   now,
		Projects:  []string{"FN", "AD", "MZN"},
	}
}

// GenerateMockDeployments generates mock deployment data
func (m *MockDataProvider) GenerateMockDeployments(ctx context.Context, config MockDataConfig) error {
	if config.Count <= 0 {
		config.Count = 10
	}

	if config.StartDate.IsZero() {
		config.StartDate = time.Now().AddDate(0, -1, 0)
	}

	if config.EndDate.IsZero() {
		config.EndDate = time.Now()
	}

	if len(config.Projects) == 0 {
		config.Projects = []string{"FN", "AD", "MZN"}
	}

	environments := []domain.Environment{
		domain.EnvironmentProduction,
		domain.EnvironmentStaging,
		domain.EnvironmentQA,
		domain.EnvironmentDevelopment,
	}

	deployedByOptions := []string{
		"github-actions",
		"jenkins",
		"ci-pipeline",
		"manual-deploy",
		"auto-deploy",
	}

	statuses := []domain.DeploymentStatus{
		domain.DeploymentStatusSuccessful,
		domain.DeploymentStatusSuccessful,
		domain.DeploymentStatusSuccessful, // Weight towards successful
		domain.DeploymentStatusFailed,
		domain.DeploymentStatusRolledBack,
	}

	for i := 0; i < config.Count; i++ {
		// Generate random timestamp within range
		deployedAt := m.randomTimeInRange(config.StartDate, config.EndDate)

		// Generate task keys
		project := config.Projects[m.rand.Intn(len(config.Projects))]
		taskCount := m.rand.Intn(5) + 1 // 1-5 tasks per deployment
		taskKeys := m.generateTaskKeys(project, taskCount)

		// Generate version
		version := m.generateVersion(deployedAt)

		// Create deployment
		deployment, err := domain.NewDeployment(
			taskKeys,
			environments[m.rand.Intn(len(environments))],
			version,
		)
		if err != nil {
			return fmt.Errorf("failed to create mock deployment: %w", err)
		}

		// Set additional fields
		deployment.DeployedAt = deployedAt
		deployment.SetDeployedBy(deployedByOptions[m.rand.Intn(len(deployedByOptions))])
		deployment.SetCommitSHA(m.generateCommitSHA())
		deployment.SetStatus(statuses[m.rand.Intn(len(statuses))])

		// Add metadata
		metadata := &domain.DeploymentMetadata{
			PipelineID:                fmt.Sprintf("%d", m.rand.Intn(100000)),
			DeploymentDurationSeconds: m.rand.Intn(600) + 60, // 1-10 minutes
		}

		if deployment.DeployedBy == "github-actions" {
			metadata.PipelineURL = fmt.Sprintf("https://github.com/org/repo/actions/runs/%s", metadata.PipelineID)
		}

		if deployment.Status == domain.DeploymentStatusRolledBack {
			rollbackFrom := fmt.Sprintf("v%d.%d.%d", m.rand.Intn(3), m.rand.Intn(10), m.rand.Intn(20))
			metadata.RollbackFrom = &rollbackFrom
		}

		deployment.SetMetadata(metadata)

		// Save deployment
		if err := m.repo.Save(ctx, deployment); err != nil {
			return fmt.Errorf("failed to save mock deployment: %w", err)
		}
	}

	return nil
}

// randomTimeInRange generates a random time within the given range
func (m *MockDataProvider) randomTimeInRange(start, end time.Time) time.Time {
	delta := end.Unix() - start.Unix()
	sec := m.rand.Int63n(delta) + start.Unix()
	return time.Unix(sec, 0)
}

// generateTaskKeys generates mock task keys
func (m *MockDataProvider) generateTaskKeys(project string, count int) []string {
	keys := make([]string, count)
	for i := 0; i < count; i++ {
		keys[i] = fmt.Sprintf("%s-%d", project, m.rand.Intn(10000))
	}
	return keys
}

// generateVersion generates a semantic version string
func (m *MockDataProvider) generateVersion(deployedAt time.Time) string {
	// Use date-based versioning with some randomness
	major := deployedAt.Year() - 2023 // Start from 2023
	minor := int(deployedAt.Month())
	patch := m.rand.Intn(100)

	// Sometimes add a pre-release tag
	if m.rand.Float32() < 0.2 {
		return fmt.Sprintf("v%d.%d.%d-rc%d", major, minor, patch, m.rand.Intn(5)+1)
	}

	return fmt.Sprintf("v%d.%d.%d", major, minor, patch)
}

// generateCommitSHA generates a mock git commit SHA
func (m *MockDataProvider) generateCommitSHA() string {
	const charset = "abcdef0123456789"
	sha := make([]byte, 7) // Short SHA
	for i := range sha {
		sha[i] = charset[m.rand.Intn(len(charset))]
	}
	return string(sha)
}

// GenerateSampleMockFile generates a sample mock deployments JSON file
func GenerateSampleMockFile(ctx context.Context, filepath string) error {
	// Create a temporary repository for generating the sample
	repo := NewJSONRepository(JSONRepositoryConfig{
		Directory: ".",
		Filename:  filepath,
	})

	provider := NewMockDataProvider(repo)

	// Generate sample deployments
	sampleConfig := MockDataConfig{
		Count:     5,
		StartDate: time.Now().AddDate(0, 0, -7), // Last 7 days
		EndDate:   time.Now(),
		Projects:  []string{"FN", "AD"},
	}

	return provider.GenerateMockDeployments(ctx, sampleConfig)
}
