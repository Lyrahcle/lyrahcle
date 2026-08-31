import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECKER = ROOT / 'scripts/check-keyprovider-kid.py'

class KeyProviderKIDContractTests(unittest.TestCase):
    def test_repository_contract_passes(self):
        result = subprocess.run(['python3', str(CHECKER), str(ROOT)], text=True, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('keyprovider/KID contract: OK', result.stdout)

    def test_signer_direct_private_key_access_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for rel in [
                'apps/backend/internal/keyprovider/provider.go',
                'apps/backend/internal/agentconfig/signing.go',
                'apps/backend/internal/store/agent_config.go',
                'apps/backend/migrations/0036_agent_config_signing_kid.sql',
                'apps/sdr-agent/internal/uplink/uplink.go',
                'apps/sdr-agent/internal/remoteconfig/remoteconfig.go',
                'apps/sdr-agent/internal/remoteconfig/keys.go',
                'apps/frontend/src/components/AgentRemoteConfigPanel.jsx',
            ]:
                src = ROOT / rel
                dst = root / rel
                dst.parent.mkdir(parents=True, exist_ok=True)
                dst.write_text(src.read_text())
            signer = root / 'apps/backend/internal/agentconfig/signing.go'
            signer.write_text(signer.read_text() + '\n// os.ReadFile( private key bypass\n')
            result = subprocess.run(['python3', str(CHECKER), str(root)], text=True, capture_output=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('bypasses KeyProvider', result.stdout)

if __name__ == '__main__':
    unittest.main()
