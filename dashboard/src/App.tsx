import { Routes, Route } from 'react-router-dom'
import { Layout } from './Layout'
import { Overview } from './pages/Overview'
import { Memories } from './pages/Memories'
import { Entities } from './pages/Entities'
import { Relations } from './pages/Relations'
import { Settings } from './pages/Settings'
import './App.css'

function App() {
  return (
    <Routes>
      <Route path="/" element={<Layout />}>
        <Route index element={<Overview />} />
        <Route path="memories" element={<Memories />} />
        <Route path="entities" element={<Entities />} />
        <Route path="relations" element={<Relations />} />
        <Route path="settings" element={<Settings />} />
      </Route>
    </Routes>
  )
}

export default App
