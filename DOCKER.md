# TraceVibe Docker Usage

TraceVibe is available as a Docker image that works on Windows, macOS, and Linux (x86_64, ARM64, and ARM v7).

## Quick Start

### Using Docker Hub Image

```bash
# Run TraceVibe on port 8080
docker run -p 8080:8080 your-dockerhub-username/tracevibe:latest

# Run with persistent data storage
docker run -p 8080:8080 -v tracevibe-data:/app/data your-dockerhub-username/tracevibe:latest

# Run in background with auto-restart
docker run -d --name tracevibe -p 8080:8080 -v tracevibe-data:/app/data --restart unless-stopped your-dockerhub-username/tracevibe:latest
```

### Building Locally

```bash
# Build the image
docker build -t tracevibe:latest .

# Run the locally built image
docker run -p 8080:8080 tracevibe:latest
```

## Supported Platforms

- **linux/amd64** - Intel/AMD 64-bit (most common)
- **linux/arm64** - ARM 64-bit (Apple M1/M2, modern ARM servers)
- **linux/arm/v7** - ARM 32-bit (Raspberry Pi, older ARM devices)

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PATH` | `/app/data/tracevibe.db` | SQLite database file path |
| `GIN_MODE` | `release` | Gin framework mode (release/debug) |

## Data Persistence

TraceVibe stores data in a SQLite database. To persist data between container restarts:

```bash
# Create a named volume
docker volume create tracevibe-data

# Use the volume when running
docker run -p 8080:8080 -v tracevibe-data:/app/data your-dockerhub-username/tracevibe:latest
```

## Health Checks

The Docker image includes built-in health checks:

```bash
# Check container health
docker inspect --format='{{.State.Health.Status}}' tracevibe

# View health check logs
docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' tracevibe
```

## Development

### Multi-Platform Build

To build for multiple platforms (requires Docker buildx):

```bash
# Set up buildx
docker buildx create --name tracevibe-builder --use --bootstrap

# Build and push multi-platform image
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 --tag your-dockerhub-username/tracevibe:latest --push .
```

Or use the provided script:

```bash
# Edit docker-build.sh to set your Docker Hub username
./docker-build.sh v1.0.0
```

### Testing Different Platforms

```bash
# Test specific platform (on compatible hardware)
docker run --platform linux/arm64 -p 8080:8080 your-dockerhub-username/tracevibe:latest
```

## Troubleshooting

### Container Won't Start

```bash
# Check logs
docker logs tracevibe

# Check if port is already in use
netstat -tulpn | grep :8080

# Try different port
docker run -p 8081:8080 your-dockerhub-username/tracevibe:latest
```

### Database Issues

```bash
# Reset database (removes all data)
docker volume rm tracevibe-data

# Backup database
docker run --rm -v tracevibe-data:/data -v $(pwd):/backup alpine cp /data/tracevibe.db /backup/

# Restore database
docker run --rm -v tracevibe-data:/data -v $(pwd):/backup alpine cp /backup/tracevibe.db /data/
```

### Performance Issues

```bash
# Check resource usage
docker stats tracevibe

# Limit resources
docker run -p 8080:8080 --memory=512m --cpus=1 your-dockerhub-username/tracevibe:latest
```

## Security Considerations

- The container runs as non-root user `tracevibe` (UID 1000)
- No sensitive data is exposed in environment variables
- SQLite database files should be backed up regularly
- Consider using Docker secrets for production deployments

## Production Deployment

For production use, consider:

1. **Reverse Proxy**: Use nginx or traefik in front of TraceVibe
2. **SSL/TLS**: Terminate SSL at the reverse proxy level
3. **Backups**: Regular database backups to external storage
4. **Monitoring**: Health checks and log aggregation
5. **Updates**: Use specific version tags instead of `latest`

Example with docker-compose:

```yaml
version: '3.8'
services:
  tracevibe:
    image: your-dockerhub-username/tracevibe:v1.0.0
    ports:
      - "8080:8080"
    volumes:
      - tracevibe-data:/app/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  tracevibe-data:
```

## Support

For Docker-related issues:
1. Check the [Dockerfile](./Dockerfile) for build configuration
2. Review [docker-build.sh](./docker-build.sh) for multi-platform builds
3. Use [docker-run.sh](./docker-run.sh) for simple deployments
4. Open an issue on GitHub with Docker version and platform information