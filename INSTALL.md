# TraceVibe Installation Guide

TraceVibe can be installed in multiple ways depending on your needs and platform.

## Quick Install (Recommended)

### Option 1: Install with Go

If you have Go 1.21+ installed:

```bash
go install github.com/peshwar9/statsly/rtm-system/tracevibe@latest
```

This installs TraceVibe to `$GOPATH/bin` (usually `~/go/bin`). Make sure this directory is in your PATH.

### Option 2: Download Pre-built Binary

Download the latest release for your platform:

```bash
# macOS (Intel)
curl -L https://github.com/peshwar9/statsly/releases/download/v1.0.0/tracevibe-darwin-amd64 -o tracevibe
chmod +x tracevibe
sudo mv tracevibe /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/peshwar9/statsly/releases/download/v1.0.0/tracevibe-darwin-arm64 -o tracevibe
chmod +x tracevibe
sudo mv tracevibe /usr/local/bin/

# Linux (x64)
curl -L https://github.com/peshwar9/statsly/releases/download/v1.0.0/tracevibe-linux-amd64 -o tracevibe
chmod +x tracevibe
sudo mv tracevibe /usr/local/bin/

# Windows
# Download tracevibe-windows-amd64.exe from releases page
# Add to PATH or move to C:\Windows\System32
```

### Option 3: Install Script (Unix-like systems)

```bash
curl -fsSL https://raw.githubusercontent.com/peshwar9/statsly/main/rtm-system/tracevibe/install.sh | bash
```

## Build from Source

### Prerequisites
- Go 1.21 or later
- Git

### Steps

```bash
# Clone the repository
git clone https://github.com/peshwar9/statsly.git
cd statsly/rtm-system/tracevibe

# Build the binary
go build -o tracevibe

# Install to system (optional)
sudo mv tracevibe /usr/local/bin/

# Or add to PATH
export PATH=$PATH:$(pwd)
```

## Verify Installation

```bash
# Check version
tracevibe --version

# View help
tracevibe --help
```

## First Use

### 1. Generate RTM Guidelines and Prompt

```bash
# Generate guidelines with LLM prompt
tracevibe guidelines --with-prompt

# Or save prompt to file
tracevibe guidelines --prompt-file llm-prompt.txt
```

### 2. Create RTM with LLM

1. Copy the generated prompt and guidelines
2. Provide them to an LLM (Claude, GPT-4, etc.) along with your codebase
3. Save the generated RTM JSON/YAML

### 3. Import RTM Data

```bash
tracevibe import my-project-rtm.json --project my-project
```

### 4. View in Web UI

```bash
tracevibe serve
# Open browser to http://localhost:8080
```

## Platform-Specific Notes

### macOS

If you get "cannot be opened because the developer cannot be verified":
```bash
xattr -d com.apple.quarantine tracevibe
```

### Linux

Add to PATH permanently:
```bash
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
source ~/.bashrc
```

### Windows

Add to PATH:
1. Search for "Environment Variables" in Start Menu
2. Edit "Path" variable
3. Add directory containing tracevibe.exe

## Docker Installation (Alternative)

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
RUN git clone https://github.com/peshwar9/statsly.git
WORKDIR /app/statsly/rtm-system/tracevibe
RUN go build -o tracevibe

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/statsly/rtm-system/tracevibe/tracevibe .
EXPOSE 8080
CMD ["./tracevibe", "serve"]
```

Build and run:
```bash
docker build -t tracevibe .
docker run -p 8080:8080 -v ~/.tracevibe:/root/.tracevibe tracevibe
```

## Troubleshooting

### Issue: Command not found
**Solution**: Add TraceVibe to your PATH or use full path to binary

### Issue: Permission denied
**Solution**: Make file executable with `chmod +x tracevibe`

### Issue: Port already in use
**Solution**: Use different port: `tracevibe serve --port 8081`

### Issue: Database errors
**Solution**: TraceVibe uses SQLite. Ensure write permissions in `~/.tracevibe/`

## Uninstall

### If installed with Go:
```bash
rm $(go env GOPATH)/bin/tracevibe
```

### If installed manually:
```bash
sudo rm /usr/local/bin/tracevibe
rm -rf ~/.tracevibe
```

## Support

- GitHub Issues: https://github.com/peshwar9/statsly/issues
- Documentation: https://github.com/peshwar9/statsly/tree/main/rtm-system/tracevibe

## License

MIT License - See LICENSE file for details