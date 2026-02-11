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
export PINGCLI_PINGONE_CLIENT_CREDENTIALS_SECRET="your-client-secret"

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

### Include Actual Values

Generate module with actual values from the API (for environment management):

```bash
pingcli-terraformer export \
  --include-values \
  --out ./envs/production
```

### Skip Import Blocks

For Terraform versions < 1.5 or manual import workflows:

```bash
pingcli-terraformer export \
  --skip-imports \
  --out environment.tf
```

### Skip Dependencies

Use hardcoded UUIDs instead of Terraform references (for testing):

```bash
pingcli-terraformer export \
  --skip-dependencies \
  --out standalone.tf
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
| `--out` | stdout | Output file path |
| `--module-name` | `ping-export` | Terraform module name prefix |
| `--module-dir` | `ping-export-module` | Child module directory name |
| `--include-values` | false | Populate variable values from API |
| `--include-imports` | true | Generate import blocks in root module |
| `--skip-imports` | false | Skip generating import blocks |
| `--skip-dependencies` | false | Use hardcoded UUIDs instead of references |

## Output Examples

### Generated Module Structure

**module.tf (root module):**
```hcl
module "ping-export" {
  source = "./ping-export-module"

  pingone_environment_id = ""  # TODO: Provide environment ID
  
  # Variables with empty defaults (customize per environment)
  davinci_variable_companyName_value = ""
  davinci_connection_http_base_url = ""
}
```

**imports.tf:**
```hcl
import {
  to = module.ping-export.pingone_davinci_variable.companyName
  id = "env-id/var-id"
}

import {
  to = module.ping-export.pingone_davinci_flow.registrationFlow
  id = "env-id/flow-id"
}
```

**ping-export-module/variables.tf:**
```hcl
variable "pingone_environment_id" {
  type        = string
  description = "PingOne environment ID"
}

variable "davinci_variable_companyName_value" {
  type        = string
  description = "Value for DaVinci variable: companyName"
  default     = ""
}
```

**ping-export-module/flows.tf:**
```hcl
resource "pingone_davinci_flow" "registrationFlow" {
  environment_id = var.pingone_environment_id
  name          = "registrationFlow"
  description   = "User registration flow"
  
  connection_link {
    id   = pingone_davinci_connector_instance.httpConnector.id
    name = "HTTP Connector"
  }
  
  graph_data = jsonencode({
    # ... flow configuration
  })
}
```

## Workflow: Export and Import to Terraform

1. **Export resources:**
   ```bash
   pingcli-terraformer export --out ./terraform
   ```

2. **Review generated files:**
   ```bash
   cd terraform
   ls -la
   # module.tf - root module
   # imports.tf - import blocks
   # ping-export-module/ - resource definitions
   ```

3. **Customize variables in module.tf:**
   ```hcl
   module "ping-export" {
     source = "./ping-export-module"
     
     pingone_environment_id = "your-env-id"
     davinci_variable_companyName_value = "Acme Corp"
   }
   ```

4. **Initialize and import:**
   ```bash
   terraform init
   terraform plan  # Review planned imports
   terraform apply # Imports all resources automatically
   ```

5. **Manage with Terraform:**
   ```bash
   # Now you can manage resources with Terraform
   terraform plan
   terraform apply
   ```

## Examples

See [examples/02-full-environment-export.sh](examples/02-full-environment-export.sh) for a complete demonstration.

## PingCLI Plugin Mode

When used with PingCLI, commands are namespaced under `tf`:

```bash
# Export environment
pingcli tf export \
  --pingone-worker-environment-id "abc-123..." \
  --pingone-export-environment-id "def-456..." \
  --out environment.tf
```

## Troubleshooting

### Authentication Errors

Ensure your worker app has the following scopes:
- `p1:read:env`
- `p1:read:davinci`

### Missing Resources

The tool exports:
- DaVinci flows
- DaVinci variables
- DaVinci connector instances
- DaVinci applications
- DaVinci flow policies

Other PingOne resources (users, groups, etc.) are not included.

### Import Failures

Import blocks require Terraform 1.5+. For older versions, use `--skip-imports` and import manually:

```bash
terraform import module.ping-export.pingone_davinci_variable.var1 "env-id/var-id"
```

## Development

### Building

```bash
make build
```

### Testing

```bash
# Unit tests
make test

# Linting
make golangcilint

# Acceptance tests (requires PingOne environment)
make testacc
```

## Known Limitations

- Module variable generation is experimental - variables may show hardcoded values in some cases
- Flow JSON is exported as-is from the API; complex flows may need manual adjustment
- Connector secret properties are masked and must be manually populated

See [docs/KNOWN_LIMITATIONS.md](docs/KNOWN_LIMITATIONS.md) for details (if available).

## Version

Current version: `v0.1.0-beta.1`

Run `pingcli-terraformer --version` to check your installed version.

## References

- [PingCLI](https://github.com/pingidentity/pingcli)
- [PingOne Terraform Provider](https://github.com/pingidentity/terraform-provider-pingone)
- [PingOne DaVinci Documentation](https://docs.pingidentity.com/r/en-us/davinci/davinci_landing)

## License

[Add license information]

## Contributing

[Add contributing guidelines]
