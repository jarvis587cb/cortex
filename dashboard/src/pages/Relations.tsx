import { getTenant } from '../api'

export function Relations() {
  const { appId, externalUserId } = getTenant()

  return (
    <div>
      <h1>Relations</h1>
      <p>Tenant: <strong>{appId}</strong> / <strong>{externalUserId}</strong></p>
      <p>Relations (GET /relations, POST /relations) – UI folgt.</p>
    </div>
  )
}
