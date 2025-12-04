package azuredevops

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type ResourceType int

const (
	ResourceUnknown ResourceType = iota
	ResourceBuild
	ResourceRelease
)

type ParsedResource struct {
	Type    ResourceType
	Project string
	ID      int
}

func ParseURL(rawURL string) (*ParsedResource, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	// Check query params first for IDs
	q := u.Query()

	// Check for Build
	// URL pattern: .../{Project}/_build/results?buildId={ID}...
	if strings.Contains(u.Path, "/_build") {
		buildIdStr := q.Get("buildId")
		if buildIdStr != "" {
			bid, err := strconv.Atoi(buildIdStr)
			if err == nil {
				// Extract project
				// Split by /_build to get the prefix part
				parts := strings.Split(u.Path, "/_build")
				if len(parts) > 0 {
					// Take the part before /_build and trim trailing slash
					prefix := strings.TrimRight(parts[0], "/")
					// Split by / to get path segments
					pathParts := strings.Split(prefix, "/")
					if len(pathParts) > 0 {
						// The last segment should be the project name
						project := pathParts[len(pathParts)-1]
						// Decode project name in case it's URL encoded
						project, _ = url.QueryUnescape(project)
						
						return &ParsedResource{
							Type:    ResourceBuild,
							Project: project,
							ID:      bid,
						}, nil
					}
				}
			}
		}
	}

	// Check for Release
	// URL pattern: .../{Project}/_release?releaseId={ID}...
	// Or sometimes .../_release?_a=release-summary&releaseId={ID}
	if strings.Contains(u.Path, "/_release") {
		releaseIdStr := q.Get("releaseId")
		if releaseIdStr != "" {
			rid, err := strconv.Atoi(releaseIdStr)
			if err == nil {
				// Extract project similar to build
				parts := strings.Split(u.Path, "/_release")
				if len(parts) > 0 {
					prefix := strings.TrimRight(parts[0], "/")
					pathParts := strings.Split(prefix, "/")
					if len(pathParts) > 0 {
						project := pathParts[len(pathParts)-1]
						project, _ = url.QueryUnescape(project)

						return &ParsedResource{
							Type:    ResourceRelease,
							Project: project,
							ID:      rid,
						}, nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("could not parse build or release info from URL")
}
