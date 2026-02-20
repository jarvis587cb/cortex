import { useState } from 'react'
import { getTenant, setTenant } from '../api'

export function Settings() {
  const { appId, externalUserId } = getTenant()
  const [newAppId, setNewAppId] = useState(appId)
  const [newUserId, setNewUserId] = useState(externalUserId)
  const [saved, setSaved] = useState(false)

  const handleSave = () => {
    setTenant(newAppId, newUserId)
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  return (
    <div>
      <h1>Einstellungen</h1>
      <section style={{ marginBottom: 24 }}>
        <h2>Tenant</h2>
        <p>
          <label>App-ID: <input value={newAppId} onChange={(e) => setNewAppId(e.target.value)} /></label>
        </p>
        <p>
          <label>User-ID: <input value={newUserId} onChange={(e) => setNewUserId(e.target.value)} /></label>
        </p>
        <button type="button" onClick={handleSave}>Speichern</button>
        {saved && <span style={{ marginLeft: 8, color: 'green' }}>Gespeichert.</span>}
      </section>
      <section>
        <h2>API-Key</h2>
        <p>Optional. In lokaler Umgebung meist nicht nötig. Key wird im LocalStorage gespeichert und als Header <code>X-API-Key</code> gesendet.</p>
        <p>
          <label>API-Key: <input type="password" placeholder="leer lassen = kein Key" onChange={(e) => {
            const v = e.target.value
            if (v) localStorage.setItem('cortex_api_key', v)
            else localStorage.removeItem('cortex_api_key')
          }} /></label>
        </p>
      </section>
    </div>
  )
}
