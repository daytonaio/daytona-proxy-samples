# Daytona Proxy Server (Go)

This is example of reverse proxy for the Daytona API. It dynamically routes requests to the correct sandbox preview URL based on the request's hostname, while also handling authentication and caching.

## Features

- **Dynamic Routing**: Parses sandbox ID and port from the request's hostname (e.g., `{port}-{sandbox-id}.{domain}`)
- **Authentication**: Automatically fetches preview URL and auth token from the Daytona API and injects the `X-Daytona-Preview-Token` header
- **Smart Caching**: In-memory caching (2 minutes) to reduce latency and API load
- **Production-Ready**: Graceful shutdown and proper error handling
- **Input Validation**: Validation of sandbox IDs and ports with proper error responses
- **Health Checks**: Built-in health check endpoint at `/health`
- **Simple Configuration**: Minimal environment variables required

## Configuration

The proxy is configured using environment variables. You can place these in a `.env` file in the project root.

### Required Environment Variables

```bash
# The base URL of your Daytona API instance
DAYTONA_API_URL=https://app.daytona.io/api

# Your Daytona API key (generate from the Daytona UI)
DAYTONA_API_KEY=your-secret-api-key
```

### Optional Environment Variables

```bash
# Server port (default: 3000)
PORT=3000
```

### Setup Instructions

1. **Create the `.env` file:**

   ```sh
   cp .env.example .env
   ```

2. **Edit the `.env` file with your credentials and desired configuration**

## Running the Proxy

To run the proxy server, execute the following command in the project root:

```sh
go run main.go
```

The server will start on the port specified in your `.env` file (or default to port 3000).

## Deployment with Docker

Using Docker is the recommended way to deploy the proxy as it creates a portable, consistent, and isolated environment. Environment variables are injected at runtime for security.

### 1. Build the Docker Image

From the project root, run the following command to build the Docker image. This will create a lightweight, production-ready image named `daytona-proxy`.

```sh
docker build -t daytona-proxy .
```

### 2. Run the Docker Container

Run the container with environment variables. This will start the proxy in the background, map the internal port `3000` to the host's port `3000`, and automatically restart it if it fails.

```sh
docker run -d --restart always -p 3000:3000 \
  -e DAYTONA_API_URL="https://app.daytona.io/api" \
  -e DAYTONA_API_KEY="your-secret-api-key" \
  -e PORT="3000" \
  --name daytona-proxy daytona-proxy
```

Alternatively, you can use an environment file:

```sh
docker run -d --restart always -p 3000:3000 \
  --env-file .env \
  --name daytona-proxy daytona-proxy
```

### Managing the Container

```sh
# View logs
docker logs -f daytona-proxy

# Stop the container
docker stop daytona-proxy

# Start the container
docker start daytona-proxy

# Remove the container
docker rm daytona-proxy
```
