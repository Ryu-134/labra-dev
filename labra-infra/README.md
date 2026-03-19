# labra-infra

## Files we use most

- `env/dev/main.tf` composition
- `env/dev/variables.tf` inputs for this env
- `env/dev/terraform.tfvars` dev values
- `env/dev/outputs.tf` outputs backend/frontend should read
- `env/dev/backend.hcl.example` backend bootstrap template

## Team workflow

1. Bootstrap backend once
   - set `bootstrap_state_backend = true`
   - run `terraform init` and `terraform apply` from `env/dev`
2. Move to remote backend
   - copy `backend.hcl.example` to `backend.hcl`
   - run `terraform init -reconfigure -backend-config=backend.hcl`
3. Normal deploy flow
   - set `bootstrap_state_backend = false`
   - run `terraform plan` and `terraform apply`

