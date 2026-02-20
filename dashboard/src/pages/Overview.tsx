import { useEffect, useState } from 'react'
import { api, getTenant, type Stats, type AnalyticsData } from '../api'

export function Overview() {
  const { appId, externalUserId } = getTenant()
  const [stats, setStats] = useState<Stats | null>(null)
  const [analytics, setAnalytics] = useState<AnalyticsData | null>(null)
  const [days, setDays] = useState(30)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.getStats().then(setStats).catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    api.getAnalytics(appId, externalUserId, days).then(setAnalytics).catch((e) => setError(e.message))
  }, [appId, externalUserId, days])

  if (error) return <p style={{ color: 'red' }}>Fehler: {error}</p>

  return (
    <div>
      <h1>Übersicht</h1>
      <p>Tenant: <strong>{appId}</strong> / <strong>{externalUserId}</strong></p>

      <section style={{ marginBottom: 24 }}>
        <h2>Stats</h2>
        {stats && (
          <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
            <div style={{ padding: 16, border: '1px solid #ddd', borderRadius: 8, minWidth: 120 }}>
              <div style={{ fontSize: 24, fontWeight: 'bold' }}>{stats.memories}</div>
              <div>Memories</div>
            </div>
            <div style={{ padding: 16, border: '1px solid #ddd', borderRadius: 8, minWidth: 120 }}>
              <div style={{ fontSize: 24, fontWeight: 'bold' }}>{stats.entities}</div>
              <div>Entities</div>
            </div>
            <div style={{ padding: 16, border: '1px solid #ddd', borderRadius: 8, minWidth: 120 }}>
              <div style={{ fontSize: 24, fontWeight: 'bold' }}>{stats.relations}</div>
              <div>Relations</div>
            </div>
          </div>
        )}
      </section>

      <section>
        <h2>Analytics</h2>
        <p>
          Zeitraum:{' '}
          <button type="button" onClick={() => setDays(7)}>7 Tage</button>
          {' '}
          <button type="button" onClick={() => setDays(30)}>30 Tage</button>
        </p>
        {analytics && (
          <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 24 }}>
            <div style={{ padding: 16, border: '1px solid #ddd', borderRadius: 8 }}>
              <div style={{ fontSize: 20, fontWeight: 'bold' }}>{analytics.total_memories}</div>
              <div>Memories (Tenant)</div>
            </div>
            <div style={{ padding: 16, border: '1px solid #ddd', borderRadius: 8 }}>
              <div style={{ fontSize: 20, fontWeight: 'bold' }}>{analytics.total_bundles}</div>
              <div>Bundles</div>
            </div>
            <div style={{ padding: 16, border: '1px solid #ddd', borderRadius: 8 }}>
              <div style={{ fontSize: 20, fontWeight: 'bold' }}>{analytics.memories_with_embeddings}</div>
              <div>Mit Embeddings</div>
            </div>
          </div>
        )}
        {analytics && analytics.recent_activity.length > 0 && (
          <>
            <h3>Letzte Aktivität</h3>
            <table style={{ borderCollapse: 'collapse', width: '100%' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #ddd' }}>
                  <th style={{ textAlign: 'left', padding: 8 }}>Typ</th>
                  <th style={{ textAlign: 'left', padding: 8 }}>ID</th>
                  <th style={{ textAlign: 'left', padding: 8 }}>Zeit</th>
                </tr>
              </thead>
              <tbody>
                {analytics.recent_activity.slice(0, 20).map((a, i) => (
                  <tr key={i} style={{ borderBottom: '1px solid #eee' }}>
                    <td style={{ padding: 8 }}>{a.type}</td>
                    <td style={{ padding: 8 }}>{a.id}</td>
                    <td style={{ padding: 8 }}>{new Date(a.timestamp).toLocaleString()}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        )}
      </section>
    </div>
  )
}
