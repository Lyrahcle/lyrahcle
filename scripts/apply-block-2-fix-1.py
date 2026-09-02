from pathlib import Path

path = Path("apps/frontend/src/components/RFSpectrumWaterfall.test.tsx")
text = path.read_text(encoding="utf-8")
old = "expect(screen.getByText('1 time slice · 1 frequency · 1 sample')).toBeInTheDocument()"
new = "expect(screen.getByText('1 time slices · 1 frequencies · 1 samples')).toBeInTheDocument()"
if old not in text:
    raise SystemExit(f"Expected assertion not found in {path}; no changes made.")
path.write_text(text.replace(old, new, 1), encoding="utf-8")
print(f"Patched {path}")
