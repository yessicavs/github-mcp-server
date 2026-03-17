package github

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/github/github-mcp-server/internal/githubv4mock"
	"github.com/github/github-mcp-server/pkg/translations"
	gh "github.com/google/go-github/v82/github"
	"github.com/shurcooL/githubv4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ProjectsWrite_CreateProject(t *testing.T) {
	t.Parallel()

	toolDef := ProjectsWrite(translations.NullTranslationHelper)

	t.Run("success user project", func(t *testing.T) {
		t.Parallel()

		mockedClient := githubv4mock.NewMockedHTTPClient(
			githubv4mock.NewQueryMatcher(
				struct {
					User struct {
						ID string
					} `graphql:"user(login: $login)"`
				}{},
				map[string]any{
					"login": githubv4.String("octocat"),
				},
				githubv4mock.DataResponse(map[string]any{
					"user": map[string]any{
						"id": "U_octocat",
					},
				}),
			),
			githubv4mock.NewMutationMatcher(
				struct {
					CreateProjectV2 struct {
						ProjectV2 struct {
							ID     string
							Number int
							Title  string
							URL    string
						}
					} `graphql:"createProjectV2(input: $input)"`
				}{},
				githubv4.CreateProjectV2Input{
					OwnerID: githubv4.ID("U_octocat"),
					Title:   githubv4.String("New Project"),
				},
				nil,
				githubv4mock.DataResponse(map[string]any{
					"createProjectV2": map[string]any{
						"projectV2": map[string]any{
							"id":     "PVT_project123",
							"number": 1,
							"title":  "New Project",
							"url":    "https://github.com/users/octocat/projects/1",
						},
					},
				}),
			),
		)

		deps := BaseDeps{
			GQLClient: githubv4.NewClient(mockedClient),
		}
		handler := toolDef.Handler(deps)
		request := createMCPRequest(map[string]any{
			"method":     "create_project",
			"owner":      "octocat",
			"owner_type": "user",
			"title":      "New Project",
		})
		result, err := handler(ContextWithDeps(context.Background(), deps), &request)

		require.NoError(t, err)
		require.False(t, result.IsError)

		textContent := getTextResult(t, result)
		var response map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)
		assert.Equal(t, "PVT_project123", response["ID"])
		assert.Equal(t, float64(1), response["Number"])
	})

	t.Run("missing owner_type returns error", func(t *testing.T) {
		t.Parallel()

		deps := BaseDeps{
			GQLClient: githubv4.NewClient(githubv4mock.NewMockedHTTPClient()),
		}
		handler := toolDef.Handler(deps)
		request := createMCPRequest(map[string]any{
			"method": "create_project",
			"owner":  "octocat",
			"title":  "New Project",
		})
		result, err := handler(ContextWithDeps(context.Background(), deps), &request)

		require.NoError(t, err)
		require.True(t, result.IsError)

		textContent := getTextResult(t, result)
		assert.Contains(t, textContent.Text, "owner_type is required")
	})
}

func Test_ProjectsWrite_CreateIterationField(t *testing.T) {
	t.Parallel()

	toolDef := ProjectsWrite(translations.NullTranslationHelper)

	t.Run("success with iterations", func(t *testing.T) {
		t.Parallel()

		mockRESTClient := MockHTTPClientWithHandlers(map[string]http.HandlerFunc{
			GetOrgsProjectsV2ByProject: mockResponse(t, http.StatusOK, map[string]any{
				"id":      1,
				"node_id": "PVT_project1",
				"title":   "Org Project",
			}),
		})

		mockGQLClient := githubv4mock.NewMockedHTTPClient(
			githubv4mock.NewMutationMatcher(
				struct {
					CreateProjectV2Field struct {
						ProjectV2Field struct {
							ProjectV2IterationField struct {
								ID   string
								Name string
							} `graphql:"... on ProjectV2IterationField"`
						}
					} `graphql:"createProjectV2Field(input: $input)"`
				}{},
				githubv4.CreateProjectV2FieldInput{
					ProjectID: githubv4.ID("PVT_project1"),
					DataType:  githubv4.ProjectV2CustomFieldType("ITERATION"),
					Name:      githubv4.String("Sprint"),
				},
				nil,
				githubv4mock.DataResponse(map[string]any{
					"createProjectV2Field": map[string]any{
						"projectV2Field": map[string]any{
							"id":   "PVTIF_field1",
							"name": "Sprint",
						},
					},
				}),
			),
			githubv4mock.NewMutationMatcher(
				struct {
					UpdateProjectV2Field struct {
						ProjectV2Field struct {
							ProjectV2IterationField struct {
								ID            string
								Name          string
								Configuration struct {
									Iterations []struct {
										ID        string
										Title     string
										StartDate string
										Duration  int
									}
								}
							} `graphql:"... on ProjectV2IterationField"`
						}
					} `graphql:"updateProjectV2Field(input: $input)"`
				}{},
				UpdateProjectV2FieldInput{
					ProjectID: githubv4.ID("PVT_project1"),
					FieldID:   githubv4.ID("PVTIF_field1"),
					IterationConfiguration: &ProjectV2IterationFieldConfigurationInput{
						Duration:  githubv4.Int(7),
						StartDate: githubv4.Date{Time: time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)},
						Iterations: &[]ProjectV2IterationFieldIterationInput{
							{
								Title:     githubv4.String("Sprint 1"),
								StartDate: githubv4.Date{Time: time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)},
								Duration:  githubv4.Int(7),
							},
						},
					},
				},
				nil,
				githubv4mock.DataResponse(map[string]any{
					"updateProjectV2Field": map[string]any{
						"projectV2Field": map[string]any{
							"id":   "PVTIF_field1",
							"name": "Sprint",
							"configuration": map[string]any{
								"iterations": []any{
									map[string]any{
										"id":        "PVTI_iter1",
										"title":     "Sprint 1",
										"startDate": "2025-01-20",
										"duration":  7,
									},
								},
							},
						},
					},
				}),
			),
		)

		deps := BaseDeps{
			Client:    gh.NewClient(mockRESTClient),
			GQLClient: githubv4.NewClient(mockGQLClient),
		}
		handler := toolDef.Handler(deps)
		request := createMCPRequest(map[string]any{
			"method":         "create_iteration_field",
			"owner":          "octo-org",
			"owner_type":     "org",
			"project_number": float64(1),
			"field_name":     "Sprint",
			"duration":       float64(7),
			"start_date":     "2025-01-20",
			"iterations": []any{
				map[string]any{
					"title":     "Sprint 1",
					"startDate": "2025-01-20",
					"duration":  float64(7),
				},
			},
		})
		result, err := handler(ContextWithDeps(context.Background(), deps), &request)

		require.NoError(t, err)
		require.False(t, result.IsError)

		textContent := getTextResult(t, result)
		var response map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)
		assert.Equal(t, "PVTIF_field1", response["ID"])
	})

	t.Run("success without iterations", func(t *testing.T) {
		t.Parallel()

		mockRESTClient := MockHTTPClientWithHandlers(map[string]http.HandlerFunc{
			GetOrgsProjectsV2ByProject: mockResponse(t, http.StatusOK, map[string]any{
				"id":      1,
				"node_id": "PVT_project1",
				"title":   "Org Project",
			}),
		})

		mockGQLClient := githubv4mock.NewMockedHTTPClient(
			githubv4mock.NewMutationMatcher(
				struct {
					CreateProjectV2Field struct {
						ProjectV2Field struct {
							ProjectV2IterationField struct {
								ID   string
								Name string
							} `graphql:"... on ProjectV2IterationField"`
						}
					} `graphql:"createProjectV2Field(input: $input)"`
				}{},
				githubv4.CreateProjectV2FieldInput{
					ProjectID: githubv4.ID("PVT_project1"),
					DataType:  githubv4.ProjectV2CustomFieldType("ITERATION"),
					Name:      githubv4.String("Sprint"),
				},
				nil,
				githubv4mock.DataResponse(map[string]any{
					"createProjectV2Field": map[string]any{
						"projectV2Field": map[string]any{
							"id":   "PVTIF_field1",
							"name": "Sprint",
						},
					},
				}),
			),
			githubv4mock.NewMutationMatcher(
				struct {
					UpdateProjectV2Field struct {
						ProjectV2Field struct {
							ProjectV2IterationField struct {
								ID            string
								Name          string
								Configuration struct {
									Iterations []struct {
										ID        string
										Title     string
										StartDate string
										Duration  int
									}
								}
							} `graphql:"... on ProjectV2IterationField"`
						}
					} `graphql:"updateProjectV2Field(input: $input)"`
				}{},
				UpdateProjectV2FieldInput{
					ProjectID: githubv4.ID("PVT_project1"),
					FieldID:   githubv4.ID("PVTIF_field1"),
					IterationConfiguration: &ProjectV2IterationFieldConfigurationInput{
						Duration:  githubv4.Int(7),
						StartDate: githubv4.Date{Time: time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)},
					},
				},
				nil,
				githubv4mock.DataResponse(map[string]any{
					"updateProjectV2Field": map[string]any{
						"projectV2Field": map[string]any{
							"id":   "PVTIF_field1",
							"name": "Sprint",
							"configuration": map[string]any{
								"iterations": []any{},
							},
						},
					},
				}),
			),
		)

		deps := BaseDeps{
			Client:    gh.NewClient(mockRESTClient),
			GQLClient: githubv4.NewClient(mockGQLClient),
		}
		handler := toolDef.Handler(deps)
		request := createMCPRequest(map[string]any{
			"method":         "create_iteration_field",
			"owner":          "octo-org",
			"owner_type":     "org",
			"project_number": float64(1),
			"field_name":     "Sprint",
			"duration":       float64(7),
			"start_date":     "2025-01-20",
		})
		result, err := handler(ContextWithDeps(context.Background(), deps), &request)

		require.NoError(t, err)
		require.False(t, result.IsError)

		textContent := getTextResult(t, result)
		var response map[string]any
		err = json.Unmarshal([]byte(textContent.Text), &response)
		require.NoError(t, err)
		assert.Equal(t, "PVTIF_field1", response["ID"])
	})
}
