import { useMemo, useState } from 'react'
import { ChevronDown, Cpu, ExternalLink, Radio } from 'lucide-react'
import { useI18n } from '../lib/I18nContext.jsx'
import Panel from './Panel.jsx'
import Skeleton from './Skeleton.jsx'
import AgentKeysPanel from './AgentKeysPanel.jsx'
import AgentIdentityPanel from './AgentIdentityPanel.jsx'
import AgentRemoteConfigPanel from './AgentRemoteConfigPanel.jsx'
import TableScroll from './TableScroll.jsx'

const SDR_DEVICES = [
  {
    id: 'rtlsdr',
    name: 'RTL-SDR (incl. Blog V4)',
    driverMatch: ['rtlsdr', 'rtl-sdr', 'rtl2832'],
    docsUrl: 'https://www.rtl-sdr.com/',
  },
  {
    id: 'hackrf',
    name: 'HackRF One',
    driverMatch: ['hackrf'],
    docsUrl: 'https://hackrf.readthedocs.io/en/latest/installing_hackrf_software.html',
  },
  {
    id: 'bladerf',
    name: 'BladeRF',
    driverMatch: ['bladerf'],
    docsUrl: 'https://github.com/Nuand/bladeRF/wiki/Getting-Started%3A-Linux',
  },
  {
    id: 'uhd',
    name: 'USRP (Ettus Research, via UHD)',
    driverMatch: ['uhd', 'usrp'],
    docsUrl: 'https://files.ettus.com/manual/page_install.html',
  },
]

const RASPBERRY_PI_MODELS = ['3B+', '4B', 'Zero 2 W']

function DeviceCard({ device, detectedOn, t }) {
  const [open, setOpen] = useState(false)
  const detected = detectedOn.length > 0

  return (
    <div className="rounded-lg border border-lyra-border bg-lyra-panel/30 p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-center gap-2.5">
          <Radio className="h-4 w-4 text-lyra-gold" strokeWidth={1.75} />
          <span className="font-display text-sm font-medium text-lyra-text">{device.name}</span>
        </div>
        <span
          className={`shrink-0 rounded-full border px-2.5 py-0.5 font-mono text-[10px] uppercase tracking-wider ${
            detected
              ? 'border-lyra-good/30 bg-lyra-good/10 text-lyra-good'
              : 'border-lyra-border text-lyra-mutedDim'
          }`}
        >
          {detected ? t('connectivity.detected') : t('connectivity.supported')}
        </span>
      </div>

      {detected && (
        <p className="mt-2 font-mono text-[11px] text-lyra-mutedDim">
          {t('connectivity.detectedOn', { agents: detectedOn.join(', ') })}
        </p>
      )}

      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="mt-3 flex items-center gap-1.5 font-mono text-[11px] uppercase tracking-wider text-lyra-muted outline-none transition-colors hover:text-lyra-goldBright focus-visible:text-lyra-goldBright"
      >
        <ChevronDown className={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-180' : ''}`} strokeWidth={1.75} />
        {t('connectivity.installGuide')}
      </button>

      {open && (
        <div className="mt-2 rounded border border-lyra-borderSoft bg-lyra-bgSoft/40 p-3">
          <p className="text-xs text-lyra-muted">{t(`connectivity.guide.${device.id}`)}</p>
          <a
            href={device.docsUrl}
            className="mt-2 inline-flex items-center gap-1.5 font-mono text-[11px] text-lyra-goldBright outline-none hover:underline focus-visible:underline"
          >
            {t('connectivity.officialDocs')}
            <ExternalLink className="h-3 w-3" strokeWidth={1.75} />
          </a>
        </div>
      )}
    </div>
  )
}

export default function ConnectivityPanel({ admin, agentKeys, agentIdentities, onAgentKeysChanged, onAgentIdentitiesChanged }) {
  const { t } = useI18n()

  const detectionByDevice = useMemo(() => {
    const map = {}
    for (const device of SDR_DEVICES) map[device.id] = []
    if (!agentKeys) return map
    for (const agent of agentKeys) {
      if (agent.revoked || !agent.device) continue
      const haystack = `${agent.device.driver} ${agent.device.hardware}`.toLowerCase()
      for (const device of SDR_DEVICES) {
        if (device.driverMatch.some((m) => haystack.includes(m))) {
          map[device.id].push(agent.agent_id)
        }
      }
    }
    return map
  }, [agentKeys])

  return (
    <div className="grid gap-6">
      <Panel eyebrow={t('panels.connectivity')} title={t('connectivity.supportedDevices')} uppercase>
        <p className="mb-5 text-sm text-lyra-muted">{t('connectivity.supportedDevicesHint')}</p>
        <div className="grid gap-4 sm:grid-cols-2">
          {SDR_DEVICES.map((device) => (
            <DeviceCard key={device.id} device={device} detectedOn={detectionByDevice[device.id] ?? []} t={t} />
          ))}
        </div>
      </Panel>

      <Panel eyebrow={t('panels.connectivity')} title={t('connectivity.edgeHardware')} uppercase>
        <div className="mb-4 flex items-center gap-2.5">
          <Cpu className="h-4 w-4 text-lyra-gold" strokeWidth={1.75} />
          <span className="font-display text-sm font-medium text-lyra-text">{t('connectivity.raspberryPi')}</span>
        </div>
        <p className="mb-3 text-sm text-lyra-muted">{t('connectivity.raspberryPiHint')}</p>
        <div className="mb-4 flex flex-wrap gap-2">
          {RASPBERRY_PI_MODELS.map((model) => (
            <span
              key={model}
              className="rounded-full border border-lyra-border px-3 py-1 font-mono text-xs text-lyra-muted"
            >
              {t('connectivity.raspberryPiModel')} {model}
            </span>
          ))}
        </div>
        <a
          href="https://www.raspberrypi.com/software/"
          className="inline-flex items-center gap-1.5 font-mono text-[11px] text-lyra-goldBright outline-none hover:underline focus-visible:underline"
        >
          {t('connectivity.officialDocs')}
          <ExternalLink className="h-3 w-3" strokeWidth={1.75} />
        </a>
      </Panel>

      {admin && (
        <Panel eyebrow={t('panels.connectivity')} title={t('connectivity.mtlsIdentities')} uppercase>
          {agentIdentities === null ? <Skeleton /> : <AgentIdentityPanel agents={agentIdentities} onChanged={onAgentIdentitiesChanged} />}
        </Panel>
      )}

      {admin && (
        <Panel eyebrow={t('panels.connectivity')} title={t('connectivity.remoteConfiguration')} uppercase>
          {agentIdentities === null ? <Skeleton /> : <AgentRemoteConfigPanel agents={agentIdentities} />}
        </Panel>
      )}

      {admin && (
        <Panel eyebrow={t('panels.connectivity')} title={t('connectivity.connectedAgents')} uppercase>
          <p className="mb-4 text-sm text-lyra-muted">{t('connectivity.connectedAgentsHint')}</p>
          {agentKeys === null ? <Skeleton /> : <AgentKeysWithDevices keys={agentKeys} onChanged={onAgentKeysChanged} t={t} />}
        </Panel>
      )}
    </div>
  )
}

function AgentKeysWithDevices({ keys, onChanged, t }) {
  return (
    <div>
      <AgentKeysPanel keys={keys} onChanged={onChanged} />

      {keys.length > 0 && (
        <div className="mt-6">
          <p className="mb-2 font-mono text-[11px] uppercase tracking-wider text-lyra-gold">
            {t('connectivity.deviceReports')}
          </p>
          <TableScroll>
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b border-lyra-borderSoft">
                  <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                    {t('connectivity.agent')}
                  </th>
                  <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                    {t('connectivity.device')}
                  </th>
                  <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
                    {t('connectivity.reportedAt')}
                  </th>
                </tr>
              </thead>
              <tbody>
                {keys.map((k) => (
                  <tr key={k.agent_id} className="border-b border-lyra-borderSoft/60 last:border-0">
                    <td className="py-2.5 font-mono text-xs text-lyra-goldBright">{k.agent_id}</td>
                    <td className="py-2.5 font-mono text-xs text-lyra-text">
                      {k.device ? `${k.device.hardware || k.device.driver}` : t('connectivity.noDeviceReported')}
                    </td>
                    <td className="py-2.5 font-mono text-xs text-lyra-mutedDim">
                      {k.device ? new Date(k.device.reported_at).toLocaleString() : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
        </div>
      )}
    </div>
  )
}
