import { Routes, Route } from 'react-router'
import Home from './pages/Home'
import Auth from './pages/Auth'
import Dashboard from './pages/Dashboard'
import Billing from './pages/Billing'
import Marketplace from './pages/Marketplace'
import SSHKeys from './pages/SSHKeys'
import Workspace from './pages/Workspace'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/login" element={<Auth />} />
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/billing" element={<Billing />} />
      <Route path="/marketplace" element={<Marketplace />} />
      <Route path="/ssh-keys" element={<SSHKeys />} />
      <Route path="/workspace/:id" element={<Workspace />} />
    </Routes>
  )
}
