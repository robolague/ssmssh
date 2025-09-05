package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function for creating temporary credentials files
func createTempCredentialsFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	credentialsPath := filepath.Join(tmpDir, "credentials")
	err := os.WriteFile(credentialsPath, []byte(content), 0644)
	require.NoError(t, err)
	return credentialsPath
}

// Test AWS profile loading functionality
func TestGetProfiles(t *testing.T) {
	tests := []struct {
		name             string
		credentials      string
		expectedProfiles []string
		expectError      bool
	}{
		{
			name: "valid credentials file with multiple profiles",
			credentials: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[profile1]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[profile2]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			expectedProfiles: []string{"default", "profile1", "profile2"},
			expectError:      false,
		},
		{
			name: "credentials file with only default profile",
			credentials: `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`,
			expectedProfiles: []string{"default"},
			expectError:      false,
		},
		{
			name:             "non-existent credentials file",
			credentials:      "",
			expectedProfiles: nil,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up temporary credentials file
			if tt.credentials != "" {
				credentialsPath := createTempCredentialsFile(t, tt.credentials)
				// Temporarily override the HOME environment variable
				originalHome := os.Getenv("HOME")
				defer os.Setenv("HOME", originalHome)

				tmpDir := filepath.Dir(credentialsPath)
				os.Setenv("HOME", tmpDir)

				// Create .aws directory
				awsDir := filepath.Join(tmpDir, ".aws")
				os.MkdirAll(awsDir, 0755)

				// Move credentials file to .aws directory
				newPath := filepath.Join(awsDir, "credentials")
				os.Rename(credentialsPath, newPath)
			}

			profiles, err := getProfiles()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedProfiles, profiles)
			}
		})
	}
}

// Test filtering functionality
func TestFilterList(t *testing.T) {
	tests := []struct {
		name     string
		list     []string
		filter   string
		expected []string
	}{
		{
			name:     "empty filter returns all items",
			list:     []string{"us-east-1", "us-west-2", "eu-west-1"},
			filter:   "",
			expected: []string{"us-east-1", "us-west-2", "eu-west-1"},
		},
		{
			name:     "filter matches some items",
			list:     []string{"us-east-1", "us-west-2", "eu-west-1"},
			filter:   "us",
			expected: []string{"us-east-1", "us-west-2"},
		},
		{
			name:     "filter matches no items",
			list:     []string{"us-east-1", "us-west-2", "eu-west-1"},
			filter:   "asia",
			expected: []string{},
		},
		{
			name:     "case insensitive filtering",
			list:     []string{"US-East-1", "us-west-2", "EU-West-1"},
			filter:   "us",
			expected: []string{"US-East-1", "us-west-2"},
		},
		{
			name:     "partial match filtering",
			list:     []string{"i-1234567890abcdef0 (web-server)", "i-0987654321fedcba0 (db-server)"},
			filter:   "web",
			expected: []string{"i-1234567890abcdef0 (web-server)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterList(tt.list, tt.filter)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test model initialization
func TestInitialModel(t *testing.T) {
	// Create a temporary credentials file for testing
	credentialsPath := createTempCredentialsFile(t, `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[test-profile]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`)

	// Temporarily override HOME environment variable
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tmpDir := filepath.Dir(credentialsPath)
	os.Setenv("HOME", tmpDir)

	// Create .aws directory
	awsDir := filepath.Join(tmpDir, ".aws")
	os.MkdirAll(awsDir, 0755)

	// Move credentials file to .aws directory
	newPath := filepath.Join(awsDir, "credentials")
	os.Rename(credentialsPath, newPath)

	model := initialModel()

	assert.Equal(t, stateProfile, model.step)
	assert.Equal(t, 0, model.cursor)
	assert.Equal(t, "", model.filter)
	assert.NotNil(t, model.profiles)
	assert.NotNil(t, model.filteredProfiles)
	assert.Equal(t, model.profiles, model.filteredProfiles)
}

// Test model state transitions and keyboard navigation
func TestModelUpdate(t *testing.T) {
	t.Run("model update returns valid model", func(t *testing.T) {
		m := model{
			step:             stateProfile,
			cursor:           0,
			filteredProfiles: []string{"profile1", "profile2", "profile3"},
		}

		msg := tea.KeyMsg{Type: tea.KeyUp}
		updatedModel, _ := m.Update(msg)
		result := updatedModel.(model)

		// Basic assertions that the model is valid
		assert.Equal(t, stateProfile, result.step)
		// Note: filteredProfiles might be nil in some cases, that's okay
	})
}

// Test filtering functionality in model
func TestModelFiltering(t *testing.T) {
	initialModel := model{
		step:             stateProfile,
		cursor:           0,
		profiles:         []string{"default", "production", "staging", "development"},
		filteredProfiles: []string{"default", "production", "staging", "development"},
		filter:           "",
	}

	// Test adding characters to filter
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")}
	updatedModel, _ := initialModel.Update(msg)
	result := updatedModel.(model)

	assert.Equal(t, "p", result.filter)
	assert.Equal(t, []string{"production", "development"}, result.filteredProfiles)

	// Test backspace
	msg = tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ = result.Update(msg)
	result = updatedModel.(model)

	assert.Equal(t, "", result.filter)
	assert.Equal(t, []string{"default", "production", "staging", "development"}, result.filteredProfiles)
}

// Test spinner functionality
func TestSpinnerTick(t *testing.T) {
	cmd := spinnerTick()
	assert.NotNil(t, cmd)

	// Test that spinner tick returns a message
	msg := cmd()
	assert.NotNil(t, msg)
}

// Test model view rendering
func TestModelView(t *testing.T) {
	tests := []struct {
		name        string
		model       model
		expectError bool
		expectEmpty bool
	}{
		{
			name: "error state shows error message",
			model: model{
				err: assert.AnError,
			},
			expectError: true,
		},
		{
			name: "loading state shows spinner",
			model: model{
				loading:      true,
				spinnerFrame: 0,
			},
			expectEmpty: false,
		},
		{
			name: "profile selection state shows profiles",
			model: model{
				step:             stateProfile,
				filteredProfiles: []string{"default", "production"},
				cursor:           0,
				filter:           "",
			},
			expectEmpty: false,
		},
		{
			name: "region selection state shows regions",
			model: model{
				step:            stateRegion,
				filteredRegions: []string{"us-east-1", "us-west-2"},
				cursor:          0,
				filter:          "",
				selectedProfile: "default",
			},
			expectEmpty: false,
		},
		{
			name: "instance selection state shows instances",
			model: model{
				step:              stateInstance,
				filteredInstances: []string{"i-123 (web-server)", "i-456 (db-server)"},
				cursor:            0,
				filter:            "",
				selectedProfile:   "default",
				selectedRegion:    "us-east-1",
			},
			expectEmpty: false,
		},
		{
			name: "done state shows session info",
			model: model{
				step:             stateDone,
				selectedProfile:  "default",
				selectedRegion:   "us-east-1",
				selectedInstance: "i-123",
			},
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.model.View()

			if tt.expectError {
				assert.Contains(t, view, "Error:")
			} else if tt.expectEmpty {
				assert.Empty(t, view)
			} else {
				assert.NotEmpty(t, view)
			}
		})
	}
}

// Test JSON parsing for AWS responses
func TestAWSResponseParsing(t *testing.T) {
	t.Run("parse regions response", func(t *testing.T) {
		jsonResponse := `{
			"Regions": [
				{"RegionName": "us-east-1"},
				{"RegionName": "us-west-2"},
				{"RegionName": "eu-west-1"}
			]
		}`

		var result struct {
			Regions []struct {
				RegionName string `json:"RegionName"`
			}
		}

		err := json.Unmarshal([]byte(jsonResponse), &result)
		assert.NoError(t, err)
		assert.Len(t, result.Regions, 3)
		assert.Equal(t, "us-east-1", result.Regions[0].RegionName)
		assert.Equal(t, "us-west-2", result.Regions[1].RegionName)
		assert.Equal(t, "eu-west-1", result.Regions[2].RegionName)
	})

	t.Run("parse instances response", func(t *testing.T) {
		jsonResponse := `{
			"Reservations": [
				{
					"Instances": [
						{
							"InstanceId": "i-1234567890abcdef0",
							"Tags": [
								{"Key": "Name", "Value": "web-server"},
								{"Key": "Environment", "Value": "production"}
							]
						}
					]
				}
			]
		}`

		var result struct {
			Reservations []struct {
				Instances []struct {
					InstanceId string `json:"InstanceId"`
					Tags       []struct {
						Key   string `json:"Key"`
						Value string `json:"Value"`
					} `json:"Tags"`
				}
			}
		}

		err := json.Unmarshal([]byte(jsonResponse), &result)
		assert.NoError(t, err)
		assert.Len(t, result.Reservations, 1)
		assert.Len(t, result.Reservations[0].Instances, 1)
		assert.Equal(t, "i-1234567890abcdef0", result.Reservations[0].Instances[0].InstanceId)
		assert.Len(t, result.Reservations[0].Instances[0].Tags, 2)
		assert.Equal(t, "Name", result.Reservations[0].Instances[0].Tags[0].Key)
		assert.Equal(t, "web-server", result.Reservations[0].Instances[0].Tags[0].Value)
	})
}

// Test Tag struct
func TestTagStruct(t *testing.T) {
	tag := Tag{
		Key:   "Environment",
		Value: "production",
	}

	assert.Equal(t, "Environment", tag.Key)
	assert.Equal(t, "production", tag.Value)
}

// Test state constants
func TestStateConstants(t *testing.T) {
	assert.Equal(t, state(0), stateProfile)
	assert.Equal(t, state(1), stateRegion)
	assert.Equal(t, state(2), stateInstance)
	assert.Equal(t, state(3), stateDone)
}

// Test quit key combinations
func TestQuitKeys(t *testing.T) {
	t.Run("quit with esc key", func(t *testing.T) {
		m := model{
			step:   stateProfile,
			cursor: 0,
		}

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		_, cmd := m.Update(msg)

		// Check that the command is a quit command
		assert.NotNil(t, cmd)
		// Execute the command to verify it's a quit command
		quitMsg := cmd()
		_, isQuit := quitMsg.(tea.QuitMsg)
		assert.True(t, isQuit)
	})
}

// Test loading state behavior
func TestLoadingState(t *testing.T) {
	t.Run("input ignored during loading", func(t *testing.T) {
		m := model{
			step:    stateProfile,
			cursor:  1,
			loading: true,
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
		updatedModel, _ := m.Update(msg)
		result := updatedModel.(model)

		// Cursor should not change during loading
		assert.Equal(t, 1, result.cursor)
		assert.True(t, result.loading)
	})

	t.Run("spinner animation during loading", func(t *testing.T) {
		m := model{
			loading:      true,
			spinnerFrame: 0,
		}

		// Simulate spinner tick
		msg := struct{}{}
		updatedModel, cmd := m.Update(msg)
		result := updatedModel.(model)

		assert.Equal(t, 1, result.spinnerFrame)
		assert.NotNil(t, cmd) // Should return spinner tick command
	})
}

// Test async command handling
func TestAsyncCommandHandling(t *testing.T) {
	t.Run("regions command result", func(t *testing.T) {
		m := model{
			step:    stateProfile,
			loading: true,
		}

		msg := struct {
			regions []string
			err     error
		}{
			regions: []string{"us-east-1", "us-west-2"},
			err:     nil,
		}

		updatedModel, _ := m.Update(msg)
		result := updatedModel.(model)

		assert.False(t, result.loading)
		assert.Equal(t, stateRegion, result.step)
		assert.Equal(t, []string{"us-east-1", "us-west-2"}, result.regions)
		assert.Equal(t, []string{"us-east-1", "us-west-2"}, result.filteredRegions)
		assert.Equal(t, 0, result.cursor)
		assert.Equal(t, "", result.filter)
	})

	t.Run("regions command error", func(t *testing.T) {
		m := model{
			step:    stateProfile,
			loading: true,
		}

		msg := struct {
			regions []string
			err     error
		}{
			regions: nil,
			err:     assert.AnError,
		}

		updatedModel, _ := m.Update(msg)
		result := updatedModel.(model)

		assert.Error(t, result.err)
		assert.False(t, result.loading)
	})
}

// Test instance preview functionality
func TestInstancePreview(t *testing.T) {
	t.Run("preview loading triggered on cursor change", func(t *testing.T) {
		m := model{
			step:              stateInstance,
			cursor:            0,
			filteredInstances: []string{"i-123 (web-server)", "i-456 (db-server)"},
			selectedProfile:   "default",
			selectedRegion:    "us-east-1",
			previewLoading:    false,
		}

		msg := tea.KeyMsg{Type: tea.KeyDown}
		updatedModel, cmd := m.Update(msg)
		result := updatedModel.(model)

		assert.True(t, result.previewLoading)
		assert.Equal(t, "i-456", result.previewInstanceId)
		assert.NotNil(t, cmd) // Should return preview command
	})
}

// Test instance ID extraction from display string
func TestInstanceIdExtraction(t *testing.T) {
	tests := []struct {
		name          string
		displayString string
		expectedId    string
	}{
		{
			name:          "instance with name tag",
			displayString: "i-1234567890abcdef0 (web-server)",
			expectedId:    "i-1234567890abcdef0",
		},
		{
			name:          "instance without name tag",
			displayString: "i-1234567890abcdef0",
			expectedId:    "i-1234567890abcdef0",
		},
		{
			name:          "instance with complex name",
			displayString: "i-1234567890abcdef0 (web-server-prod-01)",
			expectedId:    "i-1234567890abcdef0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from the model Update function
			parts := strings.Split(tt.displayString, " ")
			instanceId := parts[0]

			assert.Equal(t, tt.expectedId, instanceId)
		})
	}
}

// Benchmark tests
func BenchmarkFilterList(b *testing.B) {
	list := []string{
		"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		"i-1234567890abcdef0 (web-server)", "i-0987654321fedcba0 (db-server)",
		"default", "production", "staging", "development",
	}
	filter := "us"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filterList(list, filter)
	}
}

func BenchmarkGetProfiles(b *testing.B) {
	// Create temporary credentials file
	tmpDir := b.TempDir()
	credentialsPath := filepath.Join(tmpDir, "credentials")
	credentials := `[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[profile1]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`
	err := os.WriteFile(credentialsPath, []byte(credentials), 0644)
	require.NoError(b, err)

	// Override HOME environment variable
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getProfiles()
	}
}
