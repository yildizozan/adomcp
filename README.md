# Azure DevOps MCP Server

This is an MCP server implementation in Go that connects to an on-premise Azure DevOps instance. It provides tools to fetch build and release details, including logs.

## Features

- **SSE Support**: Implements the MCP Server-Sent Events (SSE) transport.
- **Builds**: List builds, get build details, get build logs.
- **Releases**: List releases, get release details, get release logs.
- **On-Premise**: Designed to work with on-premise Azure DevOps installations.

## Prerequisites

- Go 1.20+
- Access to an Azure DevOps instance (URL and Personal Access Token)

## Installation

1. Clone the repository.
2. Build the server:
   ```bash
   go build .
   ```

## Configuration

The server is configured via environment variables. You can also create a `.env` file in the same directory as the executable.

- `ADO_URL`: The base URL of your Azure DevOps collection (e.g., `https://ado.example.com/DefaultCollection`).
- `ADO_ORG`: (Optional) Organization name if not included in the URL.
- `ADO_PROJECT`: The project name.
- `ADO_TOKEN`: Your Personal Access Token (PAT).
- `PORT`: The port to listen on (default: 8080). Can also be set via `-port` flag.

## Usage

Start the server:

```bash
export ADO_URL="https://ado.example.com/DefaultCollection"
export ADO_PROJECT="MyProject"
export ADO_TOKEN="your-pat-token"
./adomcp -port 9090
```

The server will start on port 9090 (or the configured port).

### MCP Connection

Connect your MCP client (e.g., Claude Desktop, IDE extension) to the SSE endpoint:

`http://localhost:8080/sse`

## Tools

### `list_builds`
List recent builds.
- `top` (optional): Number of builds to retrieve (default: 10).
- `project` (optional): Project name (overrides default).

### `get_build`
Get details of a specific build.
- `buildId` (required): The ID of the build.
- `project` (optional): Project name (overrides default).

### `get_build_logs`
Get logs for a specific build.
- `buildId` (required): The ID of the build.
- `project` (optional): Project name (overrides default).

### `list_releases`
List recent releases.
- `top` (optional): Number of releases to retrieve (default: 10).
- `project` (optional): Project name (overrides default).

### `get_release`
Get details of a specific release.
- `releaseId` (required): The ID of the release.
- `project` (optional): Project name (overrides default).

### `get_release_logs`
Get logs for a specific release.
- `releaseId` (required): The ID of the release.
- `project` (optional): Project name (overrides default).
# adomcp
