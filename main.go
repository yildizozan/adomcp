package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/yildizozan/adomcp/azuredevops"
	"github.com/yildizozan/adomcp/mcp"
	"log"
	"net/http"
	"os"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist, we'll use environment variables
		// But if it exists and fails, we might want to know, or just ignore.
		// Usually silent ignore or log info is fine for optional .env
	}

	var port string
	defaultPort := os.Getenv("PORT")
	if defaultPort == "" {
		defaultPort = "8080"
	}

	flag.StringVar(&port, "port", defaultPort, "Port to listen on")
	flag.Parse()

	adoURL := os.Getenv("ADO_URL")
	adoOrg := os.Getenv("ADO_ORG")
	adoProject := os.Getenv("ADO_PROJECT")
	adoToken := os.Getenv("ADO_TOKEN")

	if adoURL == "" || adoToken == "" {
		log.Fatal("ADO_URL and ADO_TOKEN environment variables are required")
	}

	client := azuredevops.NewClient(adoURL, adoOrg, adoProject, adoToken)
	server := mcp.NewServer()

	// Register list_builds
	server.RegisterTool(mcp.Tool{
		Name:        "list_builds",
		Description: "List recent builds",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"top": map[string]interface{}{
					"type": "integer",
					"description": "Number of builds to retrieve (default 10)",
				},
				"project": map[string]interface{}{
					"type": "string",
					"description": "Project name (optional, overrides default)",
				},
			},
		},
	}, func(args map[string]interface{}) (*mcp.CallToolResult, error) {
		top := 10
		if t, ok := args["top"].(float64); ok {
			top = int(t)
		}
		project, _ := args["project"].(string)
		
		builds, err := client.GetBuilds(project, top)
		if err != nil {
			return nil, err
		}
		
		buildsJSON, _ := json.MarshalIndent(builds, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: string(buildsJSON)}},
		}, nil
	})

	// Register get_build
	server.RegisterTool(mcp.Tool{
		Name:        "get_build",
		Description: "Get build details",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"buildId": map[string]interface{}{
					"type": "integer",
					"description": "ID of the build",
				},
				"project": map[string]interface{}{
					"type": "string",
					"description": "Project name (optional, overrides default)",
				},
			},
			"required": []string{"buildId"},
		},
	}, func(args map[string]interface{}) (*mcp.CallToolResult, error) {
		buildIdFloat, ok := args["buildId"].(float64)
		if !ok {
			return nil, fmt.Errorf("buildId is required and must be an integer")
		}
		buildId := int(buildIdFloat)
		project, _ := args["project"].(string)
		
		build, err := client.GetBuild(project, buildId)
		if err != nil {
			return nil, err
		}
		
		buildJSON, _ := json.MarshalIndent(build, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: string(buildJSON)}},
		}, nil
	})

	// Register get_build_logs
	server.RegisterTool(mcp.Tool{
		Name:        "get_build_logs",
		Description: "Get build logs",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"buildId": map[string]interface{}{
					"type": "integer",
					"description": "ID of the build",
				},
				"project": map[string]interface{}{
					"type": "string",
					"description": "Project name (optional, overrides default)",
				},
			},
			"required": []string{"buildId"},
		},
	}, func(args map[string]interface{}) (*mcp.CallToolResult, error) {
		buildIdFloat, ok := args["buildId"].(float64)
		if !ok {
			return nil, fmt.Errorf("buildId is required and must be an integer")
		}
		buildId := int(buildIdFloat)
		project, _ := args["project"].(string)
		
		logs, err := client.GetBuildLogs(project, buildId)
		if err != nil {
			return nil, err
		}
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: logs}},
		}, nil
	})

	// Register list_releases
	server.RegisterTool(mcp.Tool{
		Name:        "list_releases",
		Description: "List recent releases",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"top": map[string]interface{}{
					"type": "integer",
					"description": "Number of releases to retrieve (default 10)",
				},
				"project": map[string]interface{}{
					"type": "string",
					"description": "Project name (optional, overrides default)",
				},
			},
		},
	}, func(args map[string]interface{}) (*mcp.CallToolResult, error) {
		top := 10
		if t, ok := args["top"].(float64); ok {
			top = int(t)
		}
		project, _ := args["project"].(string)
		
		releases, err := client.GetReleases(project, top)
		if err != nil {
			return nil, err
		}
		
		releasesJSON, _ := json.MarshalIndent(releases, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: string(releasesJSON)}},
		}, nil
	})

	// Register get_release
	server.RegisterTool(mcp.Tool{
		Name:        "get_release",
		Description: "Get release details",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"releaseId": map[string]interface{}{
					"type": "integer",
					"description": "ID of the release",
				},
				"project": map[string]interface{}{
					"type": "string",
					"description": "Project name (optional, overrides default)",
				},
			},
			"required": []string{"releaseId"},
		},
	}, func(args map[string]interface{}) (*mcp.CallToolResult, error) {
		releaseIdFloat, ok := args["releaseId"].(float64)
		if !ok {
			return nil, fmt.Errorf("releaseId is required and must be an integer")
		}
		releaseId := int(releaseIdFloat)
		project, _ := args["project"].(string)
		
		release, err := client.GetRelease(project, releaseId)
		if err != nil {
			return nil, err
		}
		
		releaseJSON, _ := json.MarshalIndent(release, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: string(releaseJSON)}},
		}, nil
	})

	// Register get_release_logs
	server.RegisterTool(mcp.Tool{
		Name:        "get_release_logs",
		Description: "Get release logs",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"releaseId": map[string]interface{}{
					"type": "integer",
					"description": "ID of the release",
				},
				"project": map[string]interface{}{
					"type": "string",
					"description": "Project name (optional, overrides default)",
				},
			},
			"required": []string{"releaseId"},
		},
	}, func(args map[string]interface{}) (*mcp.CallToolResult, error) {
		releaseIdFloat, ok := args["releaseId"].(float64)
		if !ok {
			return nil, fmt.Errorf("releaseId is required and must be an integer")
		}
		releaseId := int(releaseIdFloat)
		project, _ := args["project"].(string)
		
		logs, err := client.GetReleaseLogs(project, releaseId)
		if err != nil {
			return nil, err
		}
		
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: logs}},
		}, nil
	})

	log.Printf("Starting MCP server on port %s...", port)
	if err := http.ListenAndServe(":"+port, server); err != nil {
		log.Fatal(err)
	}
}
