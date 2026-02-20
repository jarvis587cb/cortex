import { useEffect, useState } from 'react'
import { api, getTenant, type Memory } from '../api'

export function Memories() {
  const { appId, externalUserId } = getTenant()
  const [list, setList] = useState<Memory[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = () => {
    setLoading(true)
    api.listSeeds(appId, externalUserId)
      .then(setList)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false))
  }

  useEffect(() => { load() }, [appId, externalUserId])

  const handleDelete = (id: number) => {
    if (!confirm('Memory löschen?')) return
    api.deleteSeed(id, appId, externalUserId)
      .then(() => load())
      .catch((e) => setError(e.message))
  }

  if (error) return <p style={{ color: 'red' }}>Fehler: {error}</p>

  return (
    <div>
      <h1>Memories</h1>
      <p>Tenant: <strong>{appId}</strong> / <strong>{externalUserId}</strong></p>
      {loading ? <p>Lade…</p> : (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid #ddd' }}>
              <th style={{ textAlign: 'left', padding: 8 }}>ID</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Typ</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Inhalt</th>
              <th style={{ textAlign: 'left', padding: 8 }}>Erstellt</th>
              <th style={{ padding: 8 }}></th>
            </tr>
          </thead>
          <tbody>
            {list.map((m) => (
              <tr key={m.id} style={{ borderBottom: '1px solid #eee' }}>
                <td style={{ padding: 8 }}>{m.id}</td>
                <td style={{ padding: 8 }}>{m.type}</td>
                <td style={{ padding: 8, maxWidth: 400, overflow: 'hidden', textOverflow: 'ellipsis' }}>{m.content}</td>
                <td style={{ padding: 8 }}>{new Date(m.created_at).toLocaleString()}</td>
                <td style={{ padding: 8 }}>
                  <button type="button" onClick={() => handleDelete(m.id)}>Löschen</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {!loading && list.length === 0 && <p>Keine Memories.</p>}
    </div>
  )
}
