package azuredevops

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	BaseURL      string
	Organization string
	Project      string
	Token        string
	HTTPClient   *http.Client
}

func NewClient(baseURL, organization, project, token string) *Client {
	return &Client{
		BaseURL:      strings.TrimRight(baseURL, "/"),
		Organization: organization,
		Project:      project,
		Token:        token,
		HTTPClient:   &http.Client{},
	}
}

func (c *Client) getRequest(project, path string) (*http.Request, error) {
	// Construct URL for on-premise: https://{server}/{organization}/{project}/_apis/{area}/{resource}?api-version={version}
	
	targetProject := c.Project
	if project != "" {
		targetProject = project
	}

	fullURL := fmt.Sprintf("%s/%s/_apis/%s", c.BaseURL, targetProject, path)
	
	// Handle cases where Project is empty (org level)
	if targetProject == "" {
		fullURL = fmt.Sprintf("%s/_apis/%s", c.BaseURL, path)
	}

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(":" + c.Token))
	req.Header.Add("Authorization", "Basic "+auth)
	req.Header.Add("Content-Type", "application/json")

	return req, nil
}

func (c *Client) doRequest(req *http.Request, v interface{}) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return err
		}
	}
	return nil
}

// Build definitions
type BuildListResponse struct {
	Count int     `json:"count"`
	Value []Build `json:"value"`
}

type Build struct {
	Id          int    `json:"id"`
	BuildNumber string `json:"buildNumber"`
	Status      string `json:"status"`
	Result      string `json:"result"`
	StartTime   string `json:"startTime"`
	FinishTime  string `json:"finishTime"`
	Url         string `json:"url"`
	Definition  struct {
		Name string `json:"name"`
	} `json:"definition"`
}

func (c *Client) GetBuilds(project string, top int) ([]Build, error) {
	path := fmt.Sprintf("build/builds?api-version=6.0&$top=%d", top)
	req, err := c.getRequest(project, path)
	if err != nil {
		return nil, err
	}

	var response BuildListResponse
	if err := c.doRequest(req, &response); err != nil {
		return nil, err
	}
	return response.Value, nil
}

func (c *Client) GetBuild(project string, buildId int) (*Build, error) {
	path := fmt.Sprintf("build/builds/%d?api-version=6.0", buildId)
	req, err := c.getRequest(project, path)
	if err != nil {
		return nil, err
	}

	var build Build
	if err := c.doRequest(req, &build); err != nil {
		return nil, err
	}
	return &build, nil
}

func (c *Client) GetBuildLogs(project string, buildId int) (string, error) {
	// First get the logs metadata to find the log IDs
	path := fmt.Sprintf("build/builds/%d/logs?api-version=6.0", buildId)
	req, err := c.getRequest(project, path)
	if err != nil {
		return "", err
	}

	type LogResponse struct {
		Value []struct {
			Id  int    `json:"id"`
			Url string `json:"url"`
		} `json:"value"`
	}
	
	var logResp LogResponse
	if err := c.doRequest(req, &logResp); err != nil {
		return "", err
	}

	var fullLogs strings.Builder
	for _, logItem := range logResp.Value {
		// Fetch actual log content
		logPath := fmt.Sprintf("build/builds/%d/logs/%d?api-version=6.0", buildId, logItem.Id)
		logReq, err := c.getRequest(project, logPath)
		if err != nil {
			continue
		}
		
		resp, err := c.HTTPClient.Do(logReq)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		
		content, _ := io.ReadAll(resp.Body)
		fullLogs.WriteString(fmt.Sprintf("--- Log ID %d ---\n", logItem.Id))
		fullLogs.Write(content)
		fullLogs.WriteString("\n")
	}

	return fullLogs.String(), nil
}

// Release definitions
type ReleaseListResponse struct {
	Count int       `json:"count"`
	Value []Release `json:"value"`
}

type Release struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	CreatedOn   string `json:"createdOn"`
	Description string `json:"description"`
	ReleaseDefinition struct {
		Name string `json:"name"`
	} `json:"releaseDefinition"`
}

func (c *Client) GetReleases(project string, top int) ([]Release, error) {
	// Release API is often under vsrm subdomain for cloud, but for on-prem it might be different.
	// Usually: https://server/collection/project/_apis/release/releases
	// We'll assume the base URL structure handles the routing or we adjust the path if needed.
	// For on-prem, it is often just /_apis/release/releases
	
	path := fmt.Sprintf("release/releases?api-version=6.0&$top=%d", top)
	
	// Note: Release API might need a different base URL logic if it's strictly separated, 
	// but for on-prem single server, it's usually under the same collection.
	
	req, err := c.getRequest(project, path)
	if err != nil {
		return nil, err
	}

	var response ReleaseListResponse
	if err := c.doRequest(req, &response); err != nil {
		return nil, err
	}
	return response.Value, nil
}

func (c *Client) GetRelease(project string, releaseId int) (*Release, error) {
	path := fmt.Sprintf("release/releases/%d?api-version=6.0", releaseId)
	req, err := c.getRequest(project, path)
	if err != nil {
		return nil, err
	}

	var release Release
	if err := c.doRequest(req, &release); err != nil {
		return nil, err
	}
	return &release, nil
}

// GetReleaseLogs is more complex as it involves environments and tasks.
// Simplified version to get logs for all environments.
func (c *Client) GetReleaseLogs(project string, releaseId int) (string, error) {
	// Fetch release details to get environment IDs
	path := fmt.Sprintf("release/releases/%d?api-version=6.0", releaseId)
	req, err := c.getRequest(project, path)
	if err != nil {
		return "", err
	}
	
	// We need a more detailed struct to parse environments for logs
	type ReleaseDetail struct {
		Environments []struct {
			Id int `json:"id"`
			Name string `json:"name"`
			DeploySteps []struct {
				ReleaseDeployPhases []struct {
					DeploymentJobs []struct {
						Tasks []struct {
							Id int `json:"id"`
							Name string `json:"name"`
							LogUrl string `json:"logUrl"`
						} `json:"tasks"`
					} `json:"deploymentJobs"`
				} `json:"releaseDeployPhases"`
			} `json:"deploySteps"`
		} `json:"environments"`
	}

	var detail ReleaseDetail
	if err := c.doRequest(req, &detail); err != nil {
		return "", err
	}

	var fullLogs strings.Builder
	
	for _, env := range detail.Environments {
		fullLogs.WriteString(fmt.Sprintf("=== Environment: %s ===\n", env.Name))
		for _, step := range env.DeploySteps {
			for _, phase := range step.ReleaseDeployPhases {
				for _, job := range phase.DeploymentJobs {
					for _, task := range job.Tasks {
						if task.LogUrl == "" {
							continue
						}
						
						// The LogUrl is usually a full URL. We need to fetch it.
						// It might be absolute.
						logReq, err := http.NewRequest("GET", task.LogUrl, nil)
						if err != nil {
							continue
						}
						auth := base64.StdEncoding.EncodeToString([]byte(":" + c.Token))
						logReq.Header.Add("Authorization", "Basic "+auth)
						
						resp, err := c.HTTPClient.Do(logReq)
						if err != nil {
							continue
						}
						
						content, _ := io.ReadAll(resp.Body)
						resp.Body.Close()
						
						fullLogs.WriteString(fmt.Sprintf("--- Task: %s ---\n", task.Name))
						fullLogs.Write(content)
						fullLogs.WriteString("\n")
					}
				}
			}
		}
	}

	return fullLogs.String(), nil
}
