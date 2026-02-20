import { getTenant } from '../api'

export function Entities() {
  const { appId, externalUserId } = getTenant()

  return (
    <div>
      <h1>Entities</h1>
      <p>Tenant: <strong>{appId}</strong> / <strong>{externalUserId}</strong></p>
      <p>Entity-Liste und Facts (GET /entities, POST /entities) – UI folgt.</p>
    </div>
  )
}
