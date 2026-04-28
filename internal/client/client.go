package client

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const defaultBaseURL = "http://127.0.0.1:3000"

var (
	graphqlURL string
	authHeader string
	httpClient = &http.Client{}
)

func init() {
	if v := os.Getenv("DAGSTER_GRAPHQL_URL"); v != "" {
		graphqlURL = v
	} else {
		graphqlURL = defaultBaseURL + "/graphql"
	}
	if v := os.Getenv("DAGSTER_AUTH"); v != "" {
		authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(v))
	}
}

func SetBaseURL(url string) {
	url = strings.TrimRight(url, "/")
	if strings.HasSuffix(url, "/graphql") {
		graphqlURL = url
	} else {
		graphqlURL = url + "/graphql"
	}
}

func SetAuth(creds string) {
	authHeader = "Basic " + base64.StdEncoding.EncodeToString([]byte(creds))
}

type gqlResp struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

func Query(gql string, variables map[string]any, out any) error {
	body, _ := json.Marshal(map[string]any{
		"query":     gql,
		"variables": variables,
	})

	req, err := http.NewRequest("POST", graphqlURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	text, _ := io.ReadAll(res.Body)
	var gr gqlResp
	if err := json.Unmarshal(text, &gr); err != nil {
		return fmt.Errorf("GraphQL request failed: %d %s\n%s", res.StatusCode, res.Status, string(text))
	}

	if len(gr.Errors) > 0 {
		msgs := make([]string, len(gr.Errors))
		for i, e := range gr.Errors {
			msgs[i] = e.Message
		}
		return fmt.Errorf("GraphQL error: %s", strings.Join(msgs, "; "))
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("GraphQL request failed: %d %s", res.StatusCode, res.Status)
	}

	if len(gr.Data) == 0 {
		return errors.New("GraphQL response missing data")
	}

	return json.Unmarshal(gr.Data, out)
}
