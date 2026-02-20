import { Link, Outlet } from 'react-router-dom'

export function Layout() {
  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      <nav style={{ width: 200, padding: 16, borderRight: '1px solid #ccc', background: '#f8f9fa' }}>
        <h2 style={{ marginTop: 0 }}>Cortex</h2>
        <ul style={{ listStyle: 'none', padding: 0 }}>
          <li><Link to="/">Übersicht</Link></li>
          <li><Link to="memories">Memories</Link></li>
          <li><Link to="entities">Entities</Link></li>
          <li><Link to="relations">Relations</Link></li>
          <li><Link to="settings">Einstellungen</Link></li>
        </ul>
      </nav>
      <main style={{ flex: 1, padding: 24 }}>
        <Outlet />
      </main>
    </div>
  )
}
