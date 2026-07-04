import { Routes, Route } from 'react-router'
import Home from './pages/Home'
import Auth from './pages/Auth'
import Dashboard from './pages/Dashboard'
import Billing from './pages/Billing'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/login" element={<Auth />} />
      <Route path="/dashboard" element={<Dashboard />} />
      <Route path="/billing" element={<Billing />} />
    </Routes>
  )
}
