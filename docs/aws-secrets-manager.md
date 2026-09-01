# AWS Secrets Manager and External Secrets

Lyrahcle can synchronize application secret material from AWS Secrets Manager into the Kubernetes Secret already consumed by the backend and CVE sync workloads.

## Ownership model

Terraform creates the Secrets Manager container but never creates a secret value. External Secrets Operator runs in its own namespace with a dedicated EKS Pod Identity role. That role can read only the Lyrahcle application secret and decrypt it with the Lyrahcle data KMS key.

The application pods receive no AWS API permissions. External Secrets Operator is the only component that calls Secrets Manager.

## Secret shape

Store one JSON object in the Terraform-created application secret. Supported keys are:

| Key | Purpose |
| --- | --- |
| `DATABASE_URL` | PostgreSQL connection URI |
| `REDIS_URL` | Authenticated `rediss://` URI in production |
| `JWT_SECRET` | Active JWT signing secret |
| `JWT_SECRET_PREVIOUS` | Optional previous JWT secret during rotation |
| `SEED_ADMIN_PASSWORD` | Bootstrap administrator password |
| `SMTP_PASSWORD` | Optional SMTP password |
| `AGENT_CONFIG_SIGNER_TOKEN` | Optional remote signing-provider token |

Do not commit the JSON value, plaintext secret files, Terraform state or Kubernetes Secret exports.

## Synchronization

The AWS Helm overlay enables a namespaced `SecretStore` and `ExternalSecret`. External Secrets Operator authenticates through its own EKS Pod Identity and uses the AWS SDK default credential chain. The resulting Kubernetes Secret is `lyrahcle-app-secrets`, preserving the existing application contract.

The AWS example points at `lyrahcle-production/app`. Change the remote key when deploying another environment. The value must match the Secrets Manager secret created by the corresponding Terraform environment.

## Rotation

Update the value in Secrets Manager rather than editing the generated Kubernetes Secret. External Secrets Operator refreshes it on the configured interval. Restart or roll the application workloads when rotating values that are read only at process startup.

For JWT rotation, populate `JWT_SECRET_PREVIOUS` with the old secret before changing `JWT_SECRET`, allow the configured access-token overlap to elapse, then remove the previous value in a later rotation.

## Verification

After the AWS platform layer and Lyrahcle chart are installed:

```bash
kubectl -n external-secrets get deployment external-secrets
kubectl -n lyrahcle get secretstore lyrahcle-aws-secrets-manager
kubectl -n lyrahcle get externalsecret
kubectl -n lyrahcle describe externalsecret lyrahcle-app
```

Do not print or decode the generated Kubernetes Secret during routine verification.
