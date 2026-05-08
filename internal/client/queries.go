package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

func GetRuns(limit int, status string) ([]Run, error) {
	filterArg := ""
	if status != "" {
		filterArg = fmt.Sprintf(", filter: { statuses: [%s] }", status)
	}

	gql := fmt.Sprintf(`
		query {
			runsOrError(limit: %d%s) {
				... on Runs {
					results {
						runId jobName status startTime endTime
					}
				}
				... on InvalidPipelineRunsFilterError { message }
				... on PythonError { message }
			}
		}
	`, limit, filterArg)

	var data struct {
		RunsOrError struct {
			Results []Run   `json:"results"`
			Message *string `json:"message"`
		} `json:"runsOrError"`
	}
	if err := Query(gql, nil, &data); err != nil {
		return nil, err
	}
	if data.RunsOrError.Message != nil {
		return nil, fmt.Errorf("%s", *data.RunsOrError.Message)
	}
	return data.RunsOrError.Results, nil
}

type RunDetail struct {
	Run    RunWithStats
	Events []RunEvent
}

func GetRun(runID string) (*RunDetail, error) {
	gql := `
		query($id: ID!) {
			runOrError(runId: $id) {
				... on Run {
					runId jobName status startTime endTime
					stats { ... on RunStatsSnapshot { stepsSucceeded stepsFailed } }
					eventConnection {
						events {
							... on MessageEvent { message timestamp stepKey level }
						}
					}
				}
				... on RunNotFoundError { message }
				... on PythonError { message }
			}
		}
	`
	var data struct {
		RunOrError struct {
			RunWithStats
			EventConnection struct {
				Events []RunEvent `json:"events"`
			} `json:"eventConnection"`
			Message *string `json:"message"`
		} `json:"runOrError"`
	}
	if err := Query(gql, map[string]any{"id": runID}, &data); err != nil {
		return nil, err
	}
	if data.RunOrError.Message != nil {
		return nil, fmt.Errorf("%s", *data.RunOrError.Message)
	}
	return &RunDetail{Run: data.RunOrError.RunWithStats, Events: data.RunOrError.EventConnection.Events}, nil
}

func GetAssets() ([]AssetNode, error) {
	gql := `
		query {
			assetNodes {
				assetKey { path }
				groupName
				assetMaterializations(limit: 1) { timestamp runId }
			}
		}
	`
	var data struct {
		AssetNodes []AssetNode `json:"assetNodes"`
	}
	if err := Query(gql, nil, &data); err != nil {
		return nil, err
	}
	return data.AssetNodes, nil
}

type RunCounts struct {
	Latest *Run
	Counts map[string]int
}

func GetRunCounts() (*RunCounts, error) {
	runs, err := GetRuns(50, "")
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, r := range runs {
		counts[r.Status]++
	}
	rc := &RunCounts{Counts: counts}
	if len(runs) > 0 {
		rc.Latest = &runs[0]
	}
	return rc, nil
}

type LogFilter struct {
	Step  string
	Level string
}

func GetRunLogs(runID string, f LogFilter) ([]RunEvent, error) {
	d, err := GetRun(runID)
	if err != nil {
		return nil, err
	}
	out := []RunEvent{}
	for _, e := range d.Events {
		if e.Message == "" {
			continue
		}
		if f.Step != "" && (e.StepKey == nil || *e.StepKey != f.Step) {
			continue
		}
		if f.Level != "" && e.Level != f.Level {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

func GetAssetDetail(path []string) (*AssetDetail, error) {
	gql := `
		query($assetKey: AssetKeyInput!) {
			assetNodeOrError(assetKey: $assetKey) {
				... on AssetNode {
					assetKey { path }
					description computeKind groupName isPartitioned jobNames
					dependencyKeys { path }
					dependedByKeys { path }
					assetMaterializations(limit: 5) {
						timestamp runId
						metadataEntries {
							label description __typename
							... on IntMetadataEntry { intValue }
							... on FloatMetadataEntry { floatValue }
							... on TextMetadataEntry { text }
							... on PathMetadataEntry { path }
							... on JsonMetadataEntry { jsonString }
							... on BoolMetadataEntry { boolValue }
						}
					}
					freshnessInfo { currentMinutesLate }
					staleStatus
					staleCauses { key { path } reason category }
					kinds
					tags { key value }
				}
				... on AssetNotFoundError { __typename }
			}
		}
	`
	var raw struct {
		AssetNodeOrError json.RawMessage `json:"assetNodeOrError"`
	}
	if err := Query(gql, map[string]any{"assetKey": map[string]any{"path": path}}, &raw); err != nil {
		return nil, err
	}
	var probe struct {
		TypeName *string `json:"__typename"`
	}
	_ = json.Unmarshal(raw.AssetNodeOrError, &probe)
	if probe.TypeName != nil && *probe.TypeName == "AssetNotFoundError" {
		return nil, fmt.Errorf("Asset not found: %s", strings.Join(path, "/"))
	}
	var d AssetDetail
	if err := json.Unmarshal(raw.AssetNodeOrError, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func getRepository() (*Repository, error) {
	gql := `
		query {
			repositoriesOrError {
				... on RepositoryConnection {
					nodes { name location { name } }
				}
			}
		}
	`
	var data struct {
		RepositoriesOrError struct {
			Nodes []Repository `json:"nodes"`
		} `json:"repositoriesOrError"`
	}
	if err := Query(gql, nil, &data); err != nil {
		return nil, err
	}
	if len(data.RepositoriesOrError.Nodes) == 0 {
		return nil, fmt.Errorf("No repository found")
	}
	return &data.RepositoriesOrError.Nodes[0], nil
}

func LaunchRun(jobName string) (string, error) {
	repo, err := getRepository()
	if err != nil {
		return "", err
	}
	gql := `
		mutation($executionParams: ExecutionParams!) {
			launchRun(executionParams: $executionParams) {
				__typename
				... on LaunchRunSuccess { run { runId } }
				... on InvalidStepError { invalidStepKey }
				... on InvalidOutputError { stepKey invalidOutputName }
				... on RunConfigValidationInvalid {
					pipelineName
					errors { message reason path }
				}
				... on PipelineNotFoundError { message }
				... on RunConflict { message }
				... on UnauthorizedError { message }
				... on PythonError { message }
				... on ConflictingExecutionParamsError { message }
				... on PresetNotFoundError { message }
				... on NoModeProvidedError { message }
			}
		}
	`
	var data struct {
		LaunchRun struct {
			TypeName         string `json:"__typename"`
			Run              *struct {
				RunID string `json:"runId"`
			} `json:"run"`
			Message          *string `json:"message"`
			InvalidStepKey   *string `json:"invalidStepKey"`
			StepKey          *string `json:"stepKey"`
			InvalidOutputName *string `json:"invalidOutputName"`
			PipelineName     *string `json:"pipelineName"`
			Errors           []struct {
				Message string   `json:"message"`
				Reason  string   `json:"reason"`
				Path    []string `json:"path"`
			} `json:"errors"`
		} `json:"launchRun"`
	}
	vars := map[string]any{
		"executionParams": map[string]any{
			"selector": map[string]any{
				"jobName":                 jobName,
				"repositoryName":          repo.Name,
				"repositoryLocationName":  repo.Location.Name,
			},
			"runConfigData": map[string]any{},
		},
	}
	if err := Query(gql, vars, &data); err != nil {
		return "", err
	}
	r := data.LaunchRun
	if len(r.Errors) > 0 {
		var msgs []string
		for _, e := range r.Errors {
			path := ""
			if len(e.Path) > 0 {
				path = " at " + strings.Join(e.Path, ".")
			}
			msgs = append(msgs, fmt.Sprintf("[%s]%s: %s", e.Reason, path, e.Message))
		}
		name := ""
		if r.PipelineName != nil {
			name = *r.PipelineName
		}
		return "", fmt.Errorf("Run config invalid for %s:\n  %s", name, strings.Join(msgs, "\n  "))
	}
	if r.InvalidStepKey != nil {
		return "", fmt.Errorf("Invalid step key: %s", *r.InvalidStepKey)
	}
	if r.InvalidOutputName != nil {
		sk := ""
		if r.StepKey != nil {
			sk = *r.StepKey
		}
		return "", fmt.Errorf("Invalid output '%s' on step '%s'", *r.InvalidOutputName, sk)
	}
	if r.Message != nil {
		return "", fmt.Errorf("%s", *r.Message)
	}
	if r.Run == nil {
		return "", fmt.Errorf("launchRun: unexpected response")
	}
	return r.Run.RunID, nil
}

func TerminateRun(runID string) (string, error) {
	gql := `
		mutation($runId: String!) {
			terminateRun(runId: $runId) {
				... on TerminateRunSuccess { run { runId status } }
				... on TerminateRunFailure { message }
				... on RunNotFoundError { message }
				... on UnauthorizedError { message }
				... on PythonError { message }
			}
		}
	`
	var data struct {
		TerminateRun struct {
			Run *struct {
				RunID  string `json:"runId"`
				Status string `json:"status"`
			} `json:"run"`
			Message *string `json:"message"`
		} `json:"terminateRun"`
	}
	if err := Query(gql, map[string]any{"runId": runID}, &data); err != nil {
		return "", err
	}
	if data.TerminateRun.Message != nil {
		return "", fmt.Errorf("%s", *data.TerminateRun.Message)
	}
	if data.TerminateRun.Run == nil {
		return "", fmt.Errorf("terminateRun: unexpected response")
	}
	return data.TerminateRun.Run.Status, nil
}

func GetSensors() ([]SensorState, error) {
	repo, err := getRepository()
	if err != nil {
		return nil, err
	}
	gql := `
		query($repoSelector: RepositorySelector!) {
			sensorsOrError(repositorySelector: $repoSelector) {
				... on Sensors {
					results {
						name
						sensorState {
							name status
							nextTick { timestamp }
							ticks(limit: 3) { status timestamp error { message } }
						}
					}
				}
				... on PythonError { message }
			}
		}
	`
	var data struct {
		SensorsOrError struct {
			Results []struct {
				Name        string      `json:"name"`
				SensorState SensorState `json:"sensorState"`
			} `json:"results"`
			Message *string `json:"message"`
		} `json:"sensorsOrError"`
	}
	vars := map[string]any{
		"repoSelector": map[string]any{
			"repositoryName":         repo.Name,
			"repositoryLocationName": repo.Location.Name,
		},
	}
	if err := Query(gql, vars, &data); err != nil {
		return nil, err
	}
	if data.SensorsOrError.Message != nil {
		return nil, fmt.Errorf("%s", *data.SensorsOrError.Message)
	}
	out := make([]SensorState, len(data.SensorsOrError.Results))
	for i, r := range data.SensorsOrError.Results {
		out[i] = r.SensorState
	}
	return out, nil
}

func GetJobs() ([]Job, error) {
	repo, err := getRepository()
	if err != nil {
		return nil, err
	}
	gql := `
		query($repoSelector: RepositorySelector!) {
			repositoryOrError(repositorySelector: $repoSelector) {
				... on Repository { jobs { name description } }
				... on RepositoryNotFoundError { message }
				... on PythonError { message }
			}
		}
	`
	var data struct {
		RepositoryOrError struct {
			Jobs    []Job   `json:"jobs"`
			Message *string `json:"message"`
		} `json:"repositoryOrError"`
	}
	vars := map[string]any{
		"repoSelector": map[string]any{
			"repositoryName":         repo.Name,
			"repositoryLocationName": repo.Location.Name,
		},
	}
	if err := Query(gql, vars, &data); err != nil {
		return nil, err
	}
	if data.RepositoryOrError.Message != nil {
		return nil, fmt.Errorf("%s", *data.RepositoryOrError.Message)
	}
	return data.RepositoryOrError.Jobs, nil
}

func GetAssetGraph() ([]AssetGraphNode, error) {
	gql := `
		query {
			assetNodes {
				assetKey { path }
				dependencyKeys { path }
				dependedByKeys { path }
				groupName
			}
		}
	`
	var data struct {
		AssetNodes []AssetGraphNode `json:"assetNodes"`
	}
	if err := Query(gql, nil, &data); err != nil {
		return nil, err
	}
	return data.AssetNodes, nil
}

type ReloadResult struct {
	Status  string
	Message string
}

func ReloadLocation() (*ReloadResult, error) {
	repo, err := getRepository()
	if err != nil {
		return nil, err
	}
	gql := `
		mutation($locationName: String!) {
			reloadRepositoryLocation(repositoryLocationName: $locationName) {
				__typename
				... on WorkspaceLocationEntry { name loadStatus }
				... on ReloadNotSupported { message }
				... on RepositoryLocationNotFound { message }
				... on UnauthorizedError { message }
				... on PythonError { message }
			}
		}
	`
	var data struct {
		ReloadRepositoryLocation struct {
			TypeName   string  `json:"__typename"`
			LoadStatus *string `json:"loadStatus"`
			Message    *string `json:"message"`
		} `json:"reloadRepositoryLocation"`
	}
	if err := Query(gql, map[string]any{"locationName": repo.Location.Name}, &data); err != nil {
		return nil, err
	}
	r := data.ReloadRepositoryLocation
	if r.LoadStatus != nil {
		return &ReloadResult{Status: *r.LoadStatus}, nil
	}
	msg := ""
	if r.Message != nil {
		msg = *r.Message
	}
	return &ReloadResult{Status: "ERROR", Message: msg}, nil
}

func GetRunErrors(runID string) ([]StepFailureEvent, error) {
	gql := `
		query($id: ID!) {
			runOrError(runId: $id) {
				... on Run {
					eventConnection {
						events {
							... on ExecutionStepFailureEvent {
								stepKey
								error { message stack causes { message stack } }
							}
						}
					}
				}
				... on RunNotFoundError { message }
				... on PythonError { message }
			}
		}
	`
	var data struct {
		RunOrError struct {
			EventConnection struct {
				Events []json.RawMessage `json:"events"`
			} `json:"eventConnection"`
			Message *string `json:"message"`
		} `json:"runOrError"`
	}
	if err := Query(gql, map[string]any{"id": runID}, &data); err != nil {
		return nil, err
	}
	if data.RunOrError.Message != nil {
		return nil, fmt.Errorf("%s", *data.RunOrError.Message)
	}
	out := []StepFailureEvent{}
	for _, raw := range data.RunOrError.EventConnection.Events {
		var ev StepFailureEvent
		if err := json.Unmarshal(raw, &ev); err == nil && ev.StepKey != "" {
			out = append(out, ev)
		}
	}
	return out, nil
}

func LaunchAssetRun(assetNames []string) (string, error) {
	repo, err := getRepository()
	if err != nil {
		return "", err
	}
	gql := `
		mutation($executionParams: ExecutionParams!) {
			launchRun(executionParams: $executionParams) {
				... on LaunchRunSuccess { run { runId } }
				... on PipelineNotFoundError { message }
				... on PythonError { message }
			}
		}
	`
	var data struct {
		LaunchRun struct {
			Run *struct {
				RunID string `json:"runId"`
			} `json:"run"`
			Message *string `json:"message"`
		} `json:"launchRun"`
	}
	vars := map[string]any{
		"executionParams": map[string]any{
			"selector": map[string]any{
				"pipelineName":           "__ASSET_JOB",
				"repositoryName":         repo.Name,
				"repositoryLocationName": repo.Location.Name,
			},
			"stepKeys": assetNames,
		},
	}
	if err := Query(gql, vars, &data); err != nil {
		return "", err
	}
	if data.LaunchRun.Message != nil {
		return "", fmt.Errorf("%s", *data.LaunchRun.Message)
	}
	if data.LaunchRun.Run == nil {
		return "", fmt.Errorf("launchAssetRun: unexpected response")
	}
	return data.LaunchRun.Run.RunID, nil
}

// AssetCheckSelection identifies a single asset check on a single asset.
//
// The Dagster GraphQL ExecutionParams.assetCheckSelection field expects a
// list of { assetKey: { path: [...] }, name: "..." } entries.
type AssetCheckSelection struct {
	AssetPath []string
	CheckName string
}

// LaunchAssetCheckRun runs a set of asset checks without rematerializing
// the underlying assets. Useful for clearing red asset-check status after
// a schema fix where the asset itself doesn't need to re-run.
func LaunchAssetCheckRun(selections []AssetCheckSelection) (string, error) {
	repo, err := getRepository()
	if err != nil {
		return "", err
	}
	gql := `
		mutation($executionParams: ExecutionParams!) {
			launchRun(executionParams: $executionParams) {
				... on LaunchRunSuccess { run { runId } }
				... on PipelineNotFoundError { message }
				... on PythonError { message }
				... on InvalidStepError { invalidStepKey }
			}
		}
	`
	var data struct {
		LaunchRun struct {
			Run *struct {
				RunID string `json:"runId"`
			} `json:"run"`
			Message        *string `json:"message"`
			InvalidStepKey *string `json:"invalidStepKey"`
		} `json:"launchRun"`
	}
	checks := make([]map[string]any, 0, len(selections))
	for _, s := range selections {
		checks = append(checks, map[string]any{
			"assetKey": map[string]any{"path": s.AssetPath},
			"name":     s.CheckName,
		})
	}
	vars := map[string]any{
		"executionParams": map[string]any{
			"selector": map[string]any{
				"pipelineName":           "__ASSET_JOB",
				"repositoryName":         repo.Name,
				"repositoryLocationName": repo.Location.Name,
				"assetCheckSelection":    checks,
				// Empty asset selection — we want checks only, no
				// asset rematerialisation. Dagster requires the field
				// to be present (even empty) when assetCheckSelection
				// is provided.
				"assetSelection": []any{},
			},
		},
	}
	if err := Query(gql, vars, &data); err != nil {
		return "", err
	}
	if data.LaunchRun.InvalidStepKey != nil {
		return "", fmt.Errorf("invalid step key: %s", *data.LaunchRun.InvalidStepKey)
	}
	if data.LaunchRun.Message != nil {
		return "", fmt.Errorf("%s", *data.LaunchRun.Message)
	}
	if data.LaunchRun.Run == nil {
		return "", fmt.Errorf("LaunchAssetCheckRun: unexpected response")
	}
	return data.LaunchRun.Run.RunID, nil
}

func GetStaleAssets() ([]StaleAssetNode, error) {
	gql := `
		query {
			assetNodes {
				assetKey { path }
				groupName
				staleStatus
				staleCauses { key { path } reason category }
				assetMaterializations(limit: 1) { timestamp }
			}
		}
	`
	var data struct {
		AssetNodes []StaleAssetNode `json:"assetNodes"`
	}
	if err := Query(gql, nil, &data); err != nil {
		return nil, err
	}
	out := []StaleAssetNode{}
	for _, a := range data.AssetNodes {
		if a.StaleStatus != "FRESH" {
			out = append(out, a)
		}
	}
	return out, nil
}

type JobConfig struct {
	JobName string
	Fields  []ConfigField
}

func GetJobConfig(jobName string) (*JobConfig, error) {
	repo, err := getRepository()
	if err != nil {
		return nil, err
	}
	gql := `
		query($selector: PipelineSelector!) {
			runConfigSchemaOrError(selector: $selector) {
				... on RunConfigSchema {
					rootConfigType {
						... on CompositeConfigType {
							fields { name description isRequired configTypeKey defaultValueAsJson }
						}
					}
				}
				... on PipelineNotFoundError { message }
				... on ModeNotFoundError { message }
				... on PythonError { message }
			}
		}
	`
	var data struct {
		RunConfigSchemaOrError struct {
			RootConfigType struct {
				Fields []ConfigField `json:"fields"`
			} `json:"rootConfigType"`
			Message *string `json:"message"`
		} `json:"runConfigSchemaOrError"`
	}
	vars := map[string]any{
		"selector": map[string]any{
			"pipelineName":           jobName,
			"repositoryName":         repo.Name,
			"repositoryLocationName": repo.Location.Name,
		},
	}
	if err := Query(gql, vars, &data); err != nil {
		return nil, err
	}
	if data.RunConfigSchemaOrError.Message != nil {
		return nil, fmt.Errorf("%s", *data.RunConfigSchemaOrError.Message)
	}
	return &JobConfig{JobName: jobName, Fields: data.RunConfigSchemaOrError.RootConfigType.Fields}, nil
}
