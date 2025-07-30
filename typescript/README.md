# Daytona Proxy Server (TypeScript)

A custom proxy server for Daytona sandboxes built with TypeScript and Express. This proxy allows you to route traffic to your Daytona sandbox instances using subdomain-based routing.

## Features

- **Subdomain-based routing**: Routes requests to sandboxes based on subdomain format `{port}-{sandbox-id}.{domain}`
- **WebSocket support**: Proxies WebSocket connections for real-time applications
- **Custom error handling**: Displays user-friendly error pages when sandboxes are unavailable
- **Authentication**: Automatically handles Daytona preview tokens
- **CORS support**: Optional CORS disabling for development

## Prerequisites

- Node.js 18 or higher
- Docker (for containerized deployment)
- Daytona API key and URL

## Configuration

Create a `.env` file in the project root:
```bash
DAYTONA_API_KEY=dtn_***
DAYTONA_API_URL=https://app.daytona.io/api
```

## Docker build 

### Building the Docker Image

To build the Docker image for the Daytona proxy server:

```bash
# Build the image
docker build -t daytona-proxy-ts .

# Or with a specific tag
docker build -t daytona-proxy-ts:latest .
```

### Running the Container

Once the image is built, you can run the container:

```bash
# Basic run command
docker run -p 1234:1234 daytona-proxy-ts

# With environment variables
docker run -p 1234:1234 \
  -e DAYTONA_API_KEY=dtn_***
  daytona-proxy-ts

# Run in detached mode
docker run -d -p 1234:1234 --name daytona-proxy daytona-proxy-ts
```

### Docker Compose (Optional)

Create a `docker-compose.yml` file for easier management:

```yaml
version: '3.8'
services:
  daytona-proxy:
    build: .
    ports:
      - "1234:1234"
    environment:
      - DAYTONA_API_KEY=${DAYTONA_API_KEY}
      - DAYTONA_API_URL=${DAYTONA_API_URL}
    restart: unless-stopped
```

Then run with:
```bash
docker-compose up -d
```

### Environment Variables

The following environment variables can be set when running the container:

- `DAYTONA_API_KEY`: Your Daytona API key (required)
- `DAYTONA_API_URL`: Daytona API URL (defaults to https://app.daytona.io/api)
- `PORT`: Port to run the proxy on (defaults to 1234)

### Production Deployment

For production deployment, consider:

1. **Using a specific Node.js version**: The Dockerfile uses `node:18-alpine` for a smaller footprint
2. **Multi-stage builds**: For even smaller production images
3. **Health checks**: Add health check endpoints to your application
4. **Logging**: Configure proper logging for containerized environments
5. **Security**: Run the container as a non-root user

### Troubleshooting

- **Port conflicts**: If port 1234 is already in use, map to a different port: `docker run -p 8080:1234 daytona-proxy-ts`
- **Environment variables**: Ensure all required environment variables are set
- **Network issues**: Check if the container can reach the Daytona API endpoints 