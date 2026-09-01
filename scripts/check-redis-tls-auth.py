#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
errors = []

backend_client = (ROOT / 'apps/backend/internal/redisclient/redisclient.go').read_text()
for token in ['redis.ParseURL', 'tls.VersionTLS12', 'AppendCertsFromPEM', 'SafeTarget', 'REDIS_CA_CERT_FILE and REDIS_TLS_SERVER_NAME require rediss://']:
    if token not in backend_client:
        errors.append(f'backend Redis client missing {token}')

backend_config = (ROOT / 'apps/backend/internal/config/config.go').read_text()
for token in ['RedisCACertFile', 'RedisTLSServerName', 'redis://localhost:6379/0', 'production REDIS_URL must be a valid rediss:// URL', 'must include a non-empty Redis password or AUTH token']:
    if token not in backend_config:
        errors.append(f'backend config missing {token}')

backend_main = (ROOT / 'apps/backend/cmd/server/main.go').read_text()
for token in ['cfg.RedisCACertFile', 'cfg.RedisTLSServerName', 'redisclient.SafeTarget(cfg.RedisURL)']:
    if token not in backend_main:
        errors.append(f'backend startup missing {token}')

cve_client = (ROOT / 'apps/cve-sync/internal/redisclient/redisclient.go').read_text()
for token in ['redis.ParseURL', 'tls.VersionTLS12', 'AppendCertsFromPEM', 'SafeTarget']:
    if token not in cve_client:
        errors.append(f'CVE Redis client missing {token}')

cve_config = (ROOT / 'apps/cve-sync/internal/config/config.go').read_text()
for token in ['RedisURL', 'RedisCACertFile', 'RedisTLSServerName', 'safeEndpoint(c.RedisURL)', 'production REDIS_URL must be a valid rediss:// URL']:
    if token not in cve_config:
        errors.append(f'CVE config missing {token}')

compose = (ROOT / 'docker-compose.yml').read_text()
if 'REDIS_URL=redis://redis:6379/0' not in compose:
    errors.append('docker compose must use an explicit redis:// URI')
if 'REDIS_ADDR=redis:6379' in compose:
    errors.append('docker compose still uses legacy REDIS_ADDR')

values = (ROOT / 'deploy/helm/lyrahcle/values.yaml').read_text()
for token in ['redis:', 'caSecretName: ""', 'caSecretKey: ca.crt', 'serverName: ""']:
    if token not in values:
        errors.append(f'Helm values missing {token}')

backend_deploy = (ROOT / 'deploy/helm/lyrahcle/templates/backend-deployment.yaml').read_text()
cve_deploy = (ROOT / 'deploy/helm/lyrahcle/templates/cve-sync-deployment.yaml').read_text()
for token in ['REDIS_CA_CERT_FILE', 'REDIS_TLS_SERVER_NAME', 'redis-ca']:
    if token not in backend_deploy:
        errors.append(f'backend Helm deployment missing {token}')
    if token not in cve_deploy:
        errors.append(f'CVE Helm deployment missing {token}')
if '- name: REDIS_URL' not in cve_deploy:
    errors.append('CVE Helm deployment must consume REDIS_URL')
if '- name: REDIS_ADDR' in cve_deploy:
    errors.append('CVE Helm deployment still exposes REDIS_ADDR')

redis_tf = (ROOT / 'deploy/terraform/aws/redis.tf').read_text()
variables_tf = (ROOT / 'deploy/terraform/aws/variables.tf').read_text()
outputs_tf = (ROOT / 'deploy/terraform/aws/outputs.tf').read_text()
for token in ['transit_encryption_enabled = true', 'transit_encryption_mode    = "required"', 'auth_token                 = var.redis_auth_token']:
    if token not in redis_tf:
        errors.append(f'AWS Redis resource missing {token}')
for token in ['length(var.redis_auth_token) >= 16', 'length(var.redis_auth_token) <= 128', 'AWS ElastiCache AUTH-compatible characters']:
    if token not in variables_tf:
        errors.append(f'AWS Redis token validation missing {token}')
if 'redis_tls_endpoint_url' not in outputs_tf or 'rediss://' not in outputs_tf:
    errors.append('AWS outputs missing rediss endpoint URL')

configuration = (ROOT / 'docs/configuration.md').read_text()
for token in ['authenticated `rediss://`', '`REDIS_CA_CERT_FILE`', '`REDIS_TLS_SERVER_NAME`', 'minimum of TLS 1.2']:
    if token not in configuration:
        errors.append(f'configuration docs missing {token}')

if errors:
    print('Redis TLS/auth contract: FAIL')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)

print('Redis TLS/auth contract: OK')
