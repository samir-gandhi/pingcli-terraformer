# Acceptance Tests

Acceptance tests validate the tool against the real PingOne DaVinci API.

## Prerequisites

1. **PingOne Test Environment**: Dedicated test environment with sample resources
2. **Service Account**: OAuth client with sufficient permissions (DaVinci Admin role)
3. **Environment Variables**:

   ```bash
   # OAuth client credentials (worker app)
   export PINGCLI_PINGONE_WORKER_CLIENT_ID="your-client-id"
   export PINGCLI_PINGONE_WORKER_CLIENT_SECRET="your-client-secret"
   
   # Worker Environment: Environment where the OAuth client exists
   export PINGCLI_PINGONE_WORKER_ENVIRONMENT_ID="worker-env-id"
   
   # Export Environment: Environment containing DaVinci resources to export
   # Optional - defaults to worker environment if not specified
   export PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID="target-env-id"
   
   # Region (NA, EU, AP, or CA)
   export PINGONE_REGION="NA"
   
   # Optional: For specific flow tests
   export TEST_FLOW_ID="known-flow-id"
   
   # Optional: Enable expensive terraform apply tests
   export ENABLE_TERRAFORM_APPLY_TEST="true"
   ```

## Running Tests

```bash
# Run all acceptance tests
go test -tags=acceptance ./tests/acceptance -v

# Run specific test
go test -tags=acceptance ./tests/acceptance -run TestExportSingleFlow -v

# Tests automatically skip if credentials not set
go test -tags=acceptance ./tests/acceptance -v
# Output: SKIP: PINGONE_CLIENT_ID not set
```

## Test Data Setup

The export environment (PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID) should contain:

- At least 1 flow with nodes
- At least 1 connector instance (connection)
- At least 1 variable
- At least 1 application with flow policy

## CI/CD Integration

Add to CI pipeline (scheduled, not on every commit):
```yaml
# .github/workflows/acceptance-tests.yml
name: Acceptance Tests
on:
  schedule:
    - cron: '0 2 * * *'  # Nightly at 2 AM
  workflow_dispatch:     # Manual trigger

jobs:
  acceptance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - name: Run acceptance tests
        env:
          PINGCLI_PINGONE_WORKER_CLIENT_ID: ${{ secrets.ACCEPTANCE_CLIENT_ID }}
          PINGCLI_PINGONE_WORKER_CLIENT_SECRET: ${{ secrets.ACCEPTANCE_CLIENT_SECRET }}
          PINGCLI_PINGONE_WORKER_ENVIRONMENT_ID: ${{ secrets.ACCEPTANCE_WORKER_ENV_ID }}
          PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID: ${{ secrets.ACCEPTANCE_EXPORT_ENV_ID }}
          PINGONE_REGION: NA
        run: go test -tags=acceptance ./tests/acceptance -v
```

## When to Run

✅ **Run acceptance tests**:
- During development of API integration (Part 3)
- Before releasing new versions
- Nightly in CI/CD against test environment
- When debugging API-related issues

❌ **Don't run acceptance tests**:
- On every commit (too slow)
- In local unit test runs (separate command)
- Without proper test environment

## Troubleshooting

### Authentication Failures

- Verify client credentials are correct
- Ensure OAuth client exists in PINGCLI_PINGONE_WORKER_ENVIRONMENT_ID
- Ensure OAuth client has DaVinci Admin role in PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID
- Check region matches environment location
- Verify OAuth client has cross-environment permissions if worker and export environments differ

### Rate Limiting

- Tests include retry logic with backoff
- Reduce concurrency if hitting rate limits consistently

### Resource Not Found

- Verify TEST_FLOW_ID points to existing flow in export environment
- Check PINGCLI_PINGONE_EXPORT_ENVIRONMENT_ID contains expected resources
