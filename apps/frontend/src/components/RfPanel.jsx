import SignalBar from './SignalBar.jsx'
import TableScroll from './TableScroll.jsx'
import { useI18n } from '../lib/I18nContext.jsx'

export default function RfPanel({ readings }) {
  const { t } = useI18n()

  if (!readings.length) {
    return (
      <div className="rounded border border-dashed border-lyra-border px-5 py-10 text-center">
        <p className="text-sm text-lyra-muted">{t('rf.empty')}</p>
        <p className="mt-1 text-xs text-lyra-mutedDim">{t('rf.emptyHint')}</p>
      </div>
    )
  }

  return (
    <TableScroll>
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-lyra-borderSoft">
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              {t('rf.agent')}
            </th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              {t('rf.frequency')}
            </th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              {t('rf.signal')}
            </th>
            <th scope="col" className="pb-2 text-[11px] font-normal uppercase tracking-wider text-lyra-mutedDim">
              {t('rf.captured')}
            </th>
          </tr>
        </thead>
        <tbody>
          {readings.map((r, i) => (
            <tr key={i} className="border-b border-lyra-borderSoft/60 last:border-0">
              <td className="py-3 font-mono text-xs text-lyra-goldBright">{r.agent_id}</td>
              <td className="py-3 font-mono text-lyra-text">
                {(r.frequency_hz / 1e6).toFixed(3)} MHz
              </td>
              <td className="py-3"><SignalBar dbm={r.power_dbm} /></td>
              <td className="py-3 font-mono text-xs text-lyra-mutedDim">
                {new Date(r.captured_at).toLocaleTimeString()}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </TableScroll>
  )
}
