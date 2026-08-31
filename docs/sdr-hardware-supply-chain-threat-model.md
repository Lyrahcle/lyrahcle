# SDR hardware and firmware supply-chain threat model

Status: pre-v1.0 security design input. This model extends the Lyrahcle STRIDE analysis to the physical SDR/edge acquisition chain.

## Scope

The trust boundary starts before the SDR agent process: vendor/manufacturer, distributor, USB device, EEPROM/firmware/FPGA images, host kernel/USB stack, SDR drivers and libraries, local configuration, and the edge agent binary. RTL-SDR-class receivers, HackRF-class transceivers, SoapySDR-compatible hardware and future FPGA-backed radios are in scope.

## Threats and controls

| Surface | STRIDE concern | Example | Required/implemented control | Residual risk |
|---|---|---|---|---|
| Vendor/distributor | Spoofing, tampering | counterfeit or substituted SDR | record manufacturer/product/serial; controlled procurement; acceptance inventory | serials can be cloned |
| USB identity/EEPROM | Spoofing | altered VID/PID/serial | backend inventory comparison and compatibility policy | USB descriptors are not cryptographic identity |
| Firmware/FPGA | Tampering, elevation | malicious firmware or bitstream | pin approved hashes/releases; offline verification before deployment; future signed firmware attestation | many commodity SDRs lack secure boot |
| Kernel/USB stack | Elevation, DoS | vulnerable driver/USB parser | patched host, minimal drivers, least privilege, device allow-listing | kernel compromise defeats userspace controls |
| SDR libraries | Tampering | replaced librtlsdr/libhackrf/Soapy module | package provenance, SBOM, pinned builds, edge integrity manifest | runtime-only TOFU is weaker than measured boot |
| Agent binary/config | Tampering | changed agent or policy | SHA-256 edge integrity reporting and backend-owned baseline | attacker with host root may forge measurements |
| RF/ADC path | Spoofing | injected RF signal or replay | orbital/RF correlation, anomaly detection, multi-sensor corroboration | RF remains an untrusted physical input |
| TX-capable radios | Elevation, safety | unauthorized transmission | TX disabled by policy unless explicitly approved; least privilege and physical controls | firmware compromise may bypass userspace policy |
| Update pipeline | Tampering | compromised release artifact | signed releases, reproducible provenance target, staged rollout and rollback | signing-key compromise remains critical |

## Pre-deployment acceptance gate

1. Verify procurement source and inventory identifiers.
2. Capture expected USB descriptors, firmware/FPGA versions and hashes where available.
3. Install only pinned driver/library versions from approved sources.
4. Register the edge agent identity and confirm mTLS.
5. Establish the edge-integrity baseline only after the host is known-good.
6. Run receive-only validation against a known RF source before production use.
7. For TX-capable hardware, prove TX is disabled unless the deployment explicitly requires it.

## Incident indicators

Treat unexpected VID/PID/serial changes, firmware/FPGA version drift, new USB interfaces, driver/module changes, edge-integrity `changed`, unexplained TX capability, or persistent RF behavior inconsistent with orbital/peer sensors as compromise indicators. Preserve the device and host image before re-baselining.

## Roadmap to stronger attestation

The current integrity feature is tamper evidence with a backend-owned TOFU baseline. Stronger assurance requires TPM measured boot/IMA, signed host measurements, secure boot, signed firmware/bitstreams where supported, and provenance/SBOM verification for the complete edge image.
