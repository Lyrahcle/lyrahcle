import tempfile
import unittest
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts' / 'check-tenant-rate-limiting.py'

class TenantRateLimitingContractTests(unittest.TestCase):
    def test_current_tree_satisfies_contract(self):
        result = subprocess.run([sys.executable, str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_checker_rejects_missing_tenant_limiter(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            for rel in [
                'apps/backend/internal/ratelimit/tenant.go',
                'apps/backend/cmd/server/main.go',
                'apps/backend/cmd/server/agent_config.go',
                'apps/backend/cmd/server/agent_identity.go',
                'apps/backend/internal/config/config.go',
                'apps/backend/internal/metrics/metrics.go',
                'apps/backend/cmd/server/rate_limit_status.go',
                'apps/frontend/src/components/RateLimitPolicyPanel.tsx',
                'apps/frontend/src/components/Dashboard.tsx',
                'apps/frontend/src/lib/api.ts',
                'apps/backend/cmd/server/rate_limit_test.go',
                'apps/backend/internal/ratelimit/tenant_test.go',
            ]:
                p = fake / rel
                p.parent.mkdir(parents=True, exist_ok=True)
                p.write_text('package placeholder\n')
            result = subprocess.run([sys.executable, str(CHECK), str(fake)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)

if __name__ == '__main__':
    unittest.main()
