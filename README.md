# PingCLI Terraformer Plugin

Export PingOne DaVinci resources to Terraform HCL with automatic dependency resolution and import block generation.

## Features

- **Complete Environment Export**: Export flows, variables, connector instances, applications, and flow policies
- **Automatic Dependency Resolution**: Generates proper Terraform references between resources
- **Import Block Generation**: Automatic Terraform import blocks for existing resources (Terraform 1.5+)
- **Module Structure**: Generates reusable Terraform modules with proper variable scaffolding
- **Dual Mode Operation**: Works as standalone CLI or PingCLI plugin
- **Two-Environment Authentication**: Isolate credentials from exported resources

## Installation

### Pre-built Binaries (Recommended)

Download the latest release for your platform from [GitHub Releases](https://github.com/samir-gandhi/pingcli-plugin-terraformer/releases):

```bash
# macOS (ARM64/M1)
curl -L https://github.com/samir-gandhi/pingcli-plugin-terraformer/releases/latest/download/pingcli-terraformer_darwin_arm64 -o pingcli-terraformer
chmod +x pingcli-terraformer
sudo mv pingcli-terraformer /usr/local/bin/

# macOS (AMD64/Intel)
curl -L https://github.com/samir-gandhi/pingcli-plugin-terraformer/releases/latest/download/pingcli-terraformer_darwin_amd64 -o pingcli-terraformer
chmod +x pingcli-terraformer
sudo mv pingcli-terraformer /usr/local/bin/

# Linux (AMD64)
curl -L https://github.com/samir-gandhi/pingcli-plugin-terraformer/releases/latest/download/pingcli-terraformer_linux_amd64 -o pingcli-terraformer
chmod +x pingcli-terraformer
sudo mv pingcli-terraformer /usr/local/bin/

# Windows (AMD64) - PowerShell
Invoke-WebRequest -Uri "https://github.com/samir-gandhi/pingcli-plugin-terraformer/releases/latest/download/pingcli-terraformer_windows_amd64.exe" -OutFile "pingcli-terraformer.exe"
# Move to a directory in your PATH
```

### Homebrew (macOS/Linux)

```bash
brew tap samir-gandhi/tap
brew install pingcli-terraformer
```

### Linux Package Managers

**Debian/Ubuntu (.deb)**:
```bash
# Download the .deb package from releases
curl -LO https://github.com/samir-gandhi/pingcli-plugin-terraformer/releases/latest/download/pingcli-terraformer_0.1.0_linux_amd64.deb
sudo dpkg -i pingcli-terraformer_0.1.0_linux_amd64.deb
```

**RedHat/Fedora (.rpm)**:
```bash
# Download the .rpm package from releases
curl -LO https://github.com/samir-gandhi/pingcli-plugin-terraformer/releases/latest/download/pingcli-terraformer_0.1.0_linux_amd64.rpm
sudo rpm -i pingcli-terraformer_0.1.0_linux_amd64.rpm
```

**Alpine (.apk)**:
```bash
# Download the .apk package from releases
curl -LO https://github.com/samir-gandhi/pingcli-plugin-terraformer/releases/latest/download/pingcli-terraformer_0.1.0_linux_amd64.apk
sudo apk add --allow-untrusted pingcli-terraformer_0.1.0_linux_amd64.apk
```

### Docker

Build and run using Docker without installing locally:

**Build the image**:
```bash
# For your local platform
docker buildx build -t pingcli-terraformer:latest .

# For specific platform (e.g., ARM64/M1 Mac)
docker buildx build --platform linux/arm64 -t pingcli-terraformer:latest .

# For specific platform (e.g., AMD64/Intel)
docker buildx build --platform linux/amd64 -t pingcli-terraformer:latest .
```

**Run export with output to mounted folder**:
```bash
# Create output directory
mkdir -p ./terraform-output

# Run export with mounted volume
docker run --rm \
  -v "$(pwd)/terraform-output:/output" \
  --env-file secrets.env \
  pingcli-terraformer:latest \
  export --out=/output
```

The generated Terraform files will be available in `./terraform-output/`.

**Alternative: Run interactively**:
```bash
docker run -it --rm \
  -v "$(pwd)/terraform-output:/output" \
  --entrypoint sh \
  --env-file secrets.env \
  pingcli-terraformer:latest
```

### From Source

```bash
git clone https://github.com/samir-gandhi/pingcli-plugin-terraformer.git
cd pingcli-plugin-terraformer
make install
```

The binary will be installed as `pingcli-terraformer` in your `$GOBIN` directory.

## Prerequisites

- PingOne environment with DaVinci
- PingOne worker application with DaVinci API Read access (DaVinci Admin Read Only Role)
- Terraform 1.5+ (for import blocks)

## Configuration

### Environment Variables

Set these environment variables to avoid passing credentials via command-line flags:

```bash
# Worker environment (for authentication)
export PINGCLI_PINGONE_ENVIRONMENT_ID="abc-123-def-456..."
export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_ID="your-client-id"
export PINGCLI_PINGONE_CLIENT_CREDENTIALS_CLIENT_SECRET="your-client-secret"

# Export environment (target resources, defaults to worker environment if not set)
export PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID="target-env-id"

# Region (NA, EU, AP, CA, AU - defaults to NA)
export PINGCLI_PINGONE_REGION_CODE="NA"
```

### Two-Environment Model

The export command uses a two-environment architecture:

- **Worker Environment**: Contains OAuth2 worker app for authentication
- **Export Environment**: Target environment containing resources to export

**Benefits:**
- Isolate credentials from exported resources
- Export from multiple environments with same worker app
- Support dev/staging/prod workflows

**Single-environment convenience**: If export environment ID is not provided, it defaults to the worker environment.

## Usage

### Basic Export

Export all DaVinci resources from an environment:

```bash
# Using environment variables
pingcli-terraformer export

# Using command-line flags
pingcli-terraformer export \
  --pingone-worker-environment-id "abc-123..." \
  --pingone-export-environment-id "def-456..." \
  --pingone-worker-client-id "client-id" \
  --pingone-worker-client-secret "client-secret" \
  --pingone-region-code "NA"
```

### Module Generation (Default)

By default, exports generate a Terraform module structure:

```bash
pingcli-terraformer export

# Generates:
# .
# ├── ping-export-module
# │   ├── outputs.tf
# │   ├── pingone_davinci_application_flow_policy.tf
# │   ├── pingone_davinci_application.tf
# │   ├── pingone_davinci_connector_instance.tf
# │   ├── pingone_davinci_flow.tf
# │   ├── pingone_davinci_variable.tf
# │   ├── variables.tf
# │   └── versions.tf
# ├── ping-export-module.tf
# ├── ping-export-terraform.auto.tfvars
# └── ping-export-variables.tf

# 2 directories, 11 files
```

## Command Reference

### Export Command

```
pingcli-terraformer export [flags]
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--services` | `pingone-davinci` | Services to export (comma-separated) |
| `--pingone-worker-environment-id` | - | Worker environment ID for authentication |
| `--pingone-export-environment-id` | Worker env | Target environment ID for resource export |
| `--pingone-worker-client-id` | - | OAuth2 client ID |
| `--pingone-worker-client-secret` | - | OAuth2 client secret |
| `--pingone-region-code` | `NA` | Region: NA, EU, AP, CA, AU |
| `--out` | stdout | Output directory path |
| `--module-name` | `ping-export` | Terraform module name prefix |
| `--module-dir` | `ping-export-module` | Child module directory name |
| `--include-values` | false | Populate variable values from API |
| `--include-imports` | true | Generate import blocks in root module |
| `--skip-imports` | false | Skip generating import blocks |
| `--skip-dependencies` | false | Use hardcoded UUIDs instead of references |

### Supported Resources

The tool exports:
- DaVinci flows (including enable and deploy)
- DaVinci variables
- DaVinci connector instances
- DaVinci applications
- DaVinci flow policies

Other PingOne resources are not yet included.

### Import Failures

Import blocks require Terraform 1.5+. For older versions, use `--skip-imports` and import manually:

```bash
terraform import module.ping-export.pingone_davinci_variable.var1 "env-id/var-id"
```

## References

- [PingCLI](https://github.com/pingidentity/pingcli)
- [PingOne Terraform Provider](https://github.com/pingidentity/terraform-provider-pingone)
- [PingOne DaVinci Documentation](https://docs.pingidentity.com/r/en-us/davinci/davinci_landing)

## License

Apache License 2.0 - see [LICENSE](LICENSE) file for details.

## Contributing

[Add contributing guidelines]
