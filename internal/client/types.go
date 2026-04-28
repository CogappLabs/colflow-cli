package client

import "encoding/json"

type AssetKey struct {
	Path []string `json:"path"`
}

type Run struct {
	RunID     string  `json:"runId"`
	JobName   string  `json:"jobName"`
	Status    string  `json:"status"`
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
}

type RunStats struct {
	StepsSucceeded int `json:"stepsSucceeded"`
	StepsFailed    int `json:"stepsFailed"`
}

type RunWithStats struct {
	Run
	Stats RunStats `json:"stats"`
}

type RunEvent struct {
	Message   string  `json:"message"`
	Timestamp string  `json:"timestamp"`
	StepKey   *string `json:"stepKey"`
	Level     string  `json:"level"`
}

type Materialization struct {
	Timestamp string          `json:"timestamp"`
	RunID     string          `json:"runId"`
	Metadata  []MetadataEntry `json:"metadataEntries,omitempty"`
}

type MetadataEntry struct {
	Label       string  `json:"label"`
	Description *string `json:"description"`
	TypeName    string  `json:"__typename"`
	Text        *string `json:"text,omitempty"`
	IntValue    *int64  `json:"intValue,omitempty"`
	FloatValue  *float64 `json:"floatValue,omitempty"`
	BoolValue   *bool   `json:"boolValue,omitempty"`
	Path        *string `json:"path,omitempty"`
	JSONString  *string `json:"jsonString,omitempty"`
}

type AssetNode struct {
	AssetKey         AssetKey          `json:"assetKey"`
	GroupName        *string           `json:"groupName"`
	Materializations []Materialization `json:"assetMaterializations"`
}

type AssetDetail struct {
	AssetKey         AssetKey          `json:"assetKey"`
	Description      *string           `json:"description"`
	ComputeKind      *string           `json:"computeKind"`
	GroupName        *string           `json:"groupName"`
	IsPartitioned    bool              `json:"isPartitioned"`
	JobNames         []string          `json:"jobNames"`
	DependencyKeys   []AssetKey        `json:"dependencyKeys"`
	DependedByKeys   []AssetKey        `json:"dependedByKeys"`
	Materializations []Materialization `json:"assetMaterializations"`
	FreshnessInfo    *struct {
		CurrentMinutesLate *float64 `json:"currentMinutesLate"`
	} `json:"freshnessInfo"`
	StaleStatus string       `json:"staleStatus"`
	StaleCauses []StaleCause `json:"staleCauses"`
	Kinds       []string     `json:"kinds"`
	Tags        []KV         `json:"tags"`
}

type StaleCause struct {
	Key      AssetKey `json:"key"`
	Reason   string   `json:"reason"`
	Category string   `json:"category"`
}

type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type StaleAssetNode struct {
	AssetKey         AssetKey          `json:"assetKey"`
	GroupName        *string           `json:"groupName"`
	StaleStatus      string            `json:"staleStatus"`
	StaleCauses      []StaleCause      `json:"staleCauses"`
	Materializations []Materialization `json:"assetMaterializations"`
}

type AssetGraphNode struct {
	AssetKey       AssetKey   `json:"assetKey"`
	DependencyKeys []AssetKey `json:"dependencyKeys"`
	DependedByKeys []AssetKey `json:"dependedByKeys"`
	GroupName      *string    `json:"groupName"`
}

type Job struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type SensorTick struct {
	Status    string  `json:"status"`
	Timestamp float64 `json:"timestamp"`
	Error     *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type SensorState struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	NextTick *struct {
		Timestamp float64 `json:"timestamp"`
	} `json:"nextTick"`
	Ticks []SensorTick `json:"ticks"`
}

type StepError struct {
	Message string   `json:"message"`
	Stack   []string `json:"stack"`
	Causes  []struct {
		Message string   `json:"message"`
		Stack   []string `json:"stack"`
	} `json:"causes"`
}

type StepFailureEvent struct {
	StepKey string    `json:"stepKey"`
	Error   StepError `json:"error"`
}

type ConfigField struct {
	Name                string  `json:"name"`
	Description         *string `json:"description"`
	IsRequired          bool    `json:"isRequired"`
	ConfigTypeKey       string  `json:"configTypeKey"`
	DefaultValueAsJSON  *string `json:"defaultValueAsJson"`
}

type Repository struct {
	Name     string `json:"name"`
	Location struct {
		Name string `json:"name"`
	} `json:"location"`
}

func (m Materialization) MarshalJSON() ([]byte, error) {
	type alias Materialization
	return json.Marshal((alias)(m))
}
