import { useState } from 'react'
import { decodeFrame } from '../lib/api.js'

const PROTOCOLS = [
  { id: 'ax25', label: 'AX.25', placeholder: '82a0a4a64040609c60868298986f03f0...' },
  { id: 'ccsds', label: 'CCSDS Space Packet', placeholder: '0864c02a000b54454c454d455452592d4f4b' },
  { id: 'pus', label: 'PUS', placeholder: '0941c22b00141003190fa10007484b2d...' },
  { id: 'csp', label: 'CSP', placeholder: '5012bf0154454d503d32312e34' },
]

export default function Decoder() {
  const [protocol, setProtocol] = useState('ccsds')
  const [hex, setHex] = useState('')
  const [result, setResult] = useState(null)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  async function handleDecode(e) {
    e.preventDefault()
    setError(null)
    setResult(null)
    setLoading(true)
    try {
      const decoded = await decodeFrame(protocol, hex.trim())
      setResult(decoded)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const active = PROTOCOLS.find((p) => p.id === protocol)

  return (
    <div>
      <form onSubmit={handleDecode} className="space-y-4">
        <div className="flex gap-1.5">
          {PROTOCOLS.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => setProtocol(p.id)}
              className={`rounded border px-3 py-1.5 font-mono text-xs tracking-wider outline-none transition-colors focus-visible:ring-1 focus-visible:ring-lyra-gold ${
                protocol === p.id
                  ? 'border-lyra-gold/50 bg-lyra-gold/10 text-lyra-goldBright'
                  : 'border-lyra-border text-lyra-mutedDim hover:text-lyra-muted'
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>

        <div>
          <label htmlFor="hex-input" className="mb-1.5 block text-xs tracking-wider text-lyra-mutedDim">
            Hex frame
          </label>
          <textarea
            id="hex-input"
            value={hex}
            onChange={(e) => setHex(e.target.value)}
            placeholder={active.placeholder}
            rows={3}
            className="w-full rounded border border-lyra-border bg-lyra-bgSoft px-3 py-2 font-mono text-xs text-lyra-text outline-none focus:border-lyra-gold"
            required
          />
        </div>

        <button
          type="submit"
          disabled={loading}
          className="rounded bg-lyra-gold px-4 py-2 text-sm font-medium text-lyra-bg outline-none transition-opacity hover:opacity-90 focus-visible:ring-2 focus-visible:ring-lyra-goldBright disabled:opacity-50"
        >
          {loading ? 'Decoding…' : 'Decode'}
        </button>
      </form>

      {error && (
        <p role="alert" className="mt-4 rounded border border-lyra-bad/30 bg-lyra-bad/10 px-3 py-2 text-sm text-lyra-bad">
          {error}
        </p>
      )}

      {result && (
        <div className="mt-5 overflow-hidden rounded border border-lyra-borderSoft">
          <table className="w-full text-left text-sm">
            <tbody>
              {Object.entries(result).map(([key, value]) => (
                <tr key={key} className="border-b border-lyra-borderSoft/60 last:border-0">
                  <td className="w-1/3 bg-lyra-bgSoft/40 px-3 py-2 font-mono text-xs tracking-wide text-lyra-mutedDim">
                    {key}
                  </td>
                  <td className="px-3 py-2 font-mono text-xs text-lyra-text">{String(value)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
