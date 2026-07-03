import { Routes, Route } from 'react-router'
import Home from './pages/Home'
import Auth from './pages/Auth'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/login" element={<Auth />} />
    </Routes>
  )
}
