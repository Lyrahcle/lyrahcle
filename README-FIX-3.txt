Lyrahcle Block 1 Mission Control - fix 3

Purpose:
Make the MissionControl ground-station test resilient to legitimate repeated
rendering of the same station identifier after RF evidence was added.

Apply from repository root:
  python3 scripts/apply-mission-control-fix-3.py

Then test:
  cd apps/frontend
  npm run typecheck
  npm test
  npm run build
