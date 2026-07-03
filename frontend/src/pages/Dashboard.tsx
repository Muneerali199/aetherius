import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import FluidBackground from '@/components/effects/FluidBackground'
import { Button } from '@/components/ui/button'

interface UserData {
  user_id: string
  email: string
  name: string
  avatar_url: string | null
  created_at: string
}

export default function Dashboard() {
  const [user, setUser] = useState<UserData | null>(null)
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()

  useEffect(() => {
    const token = localStorage.getItem('access_token')
    if (!token) {
      navigate('/login')
      return
    }

    fetch('/v1/auth/me', {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then((res) => {
        if (!res.ok) throw new Error('unauthorized')
        return res.json()
      })
      .then((data) => setUser(data))
      .catch(() => {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        navigate('/login')
      })
      .finally(() => setLoading(false))
  }, [navigate])

  const handleLogout = () => {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    navigate('/login')
  }

  if (loading) {
    return (
      <div className="relative flex min-h-screen items-center justify-center" style={{ backgroundColor: 'var(--abyssal-black)' }}>
        <FluidBackground />
        <div className="relative flex items-center gap-3">
          <span className="h-2 w-2 animate-pulse rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
          <span className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>LOADING NETWORK...</span>
        </div>
      </div>
    )
  }

  if (!user) return null

  const createdDate = new Date(user.created_at).toLocaleDateString('en-US', {
    year: 'numeric', month: 'long', day: 'numeric',
  })

  return (
    <div className="relative min-h-screen" style={{ backgroundColor: 'var(--abyssal-black)' }}>
      <FluidBackground />

      <nav className="fixed top-0 left-0 right-0 z-50" style={{ backdropFilter: 'blur(20px)', borderBottom: '1px solid rgba(0, 119, 182, 0.15)', backgroundColor: 'rgba(0, 13, 29, 0.85)' }}>
        <div className="mx-auto flex max-w-[1400px] items-center justify-between px-6 py-4 md:px-12">
          <a href="/" className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-full" style={{ border: '1px solid rgba(144, 224, 239, 0.4)' }}>
              <div className="h-2 w-2 rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
            </div>
            <span className="font-mono text-sm tracking-[0.15em]" style={{ color: 'var(--surface-mist)' }}>
              AETHERIUS
            </span>
          </a>

          <div className="flex items-center gap-4">
            <span className="hidden font-mono text-xs tracking-[0.1em] md:block" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>
              {user.email}
            </span>
            <button
              onClick={handleLogout}
              className="rounded-full px-4 py-1.5 font-mono text-xs tracking-wider transition-all duration-300 hover:brightness-110"
              style={{
                border: '1px solid rgba(255, 107, 107, 0.3)',
                color: 'rgba(255, 107, 107, 0.8)',
              }}
            >
              Disconnect
            </button>
          </div>
        </div>
      </nav>

      <div className="relative px-6 pt-32 pb-20 md:px-12">
        <div className="mx-auto max-w-[1200px]">
          <div className="mb-12">
            <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>
              NODE DASHBOARD
            </p>
            <h1 className="mt-2 font-sans text-[clamp(2rem,4vw,3.5rem)] font-semibold leading-[1.1] tracking-[-0.02em]" style={{ color: 'var(--glacial-cyan)' }}>
              Welcome, {user.name}
            </h1>
          </div>

          <div className="grid gap-6 md:grid-cols-3">
            <div className="rounded-xl p-6" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
              <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>NODE ID</p>
              <p className="mt-2 font-mono text-sm" style={{ color: 'var(--surface-mist)' }}>{user.user_id.slice(0, 8)}...</p>
            </div>

            <div className="rounded-xl p-6" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
              <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>EMAIL</p>
              <p className="mt-2 font-mono text-sm" style={{ color: 'var(--surface-mist)' }}>{user.email}</p>
            </div>

            <div className="rounded-xl p-6" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
              <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>MEMBER SINCE</p>
              <p className="mt-2 font-mono text-sm" style={{ color: 'var(--surface-mist)' }}>{createdDate}</p>
            </div>
          </div>

          <div className="mt-12 rounded-xl p-8 text-center" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
            <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full" style={{ border: '1px solid rgba(144, 224, 239, 0.2)' }}>
              <div className="h-3 w-3 rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
            </div>
            <h2 className="mt-4 font-mono text-sm tracking-[0.15em]" style={{ color: 'var(--glacial-cyan)' }}>
              CONNECT A NODE
            </h2>
            <p className="mx-auto mt-2 max-w-md text-sm leading-relaxed" style={{ color: 'rgba(202, 240, 248, 0.6)' }}>
              Deploy your first compute node to start earning. Download the agent and run the one-line install command on any Linux, macOS, or Windows machine.
            </p>
            <Button
              disabled
              className="mt-6 h-10 font-mono text-xs tracking-[0.15em] transition-all duration-300 hover:brightness-110"
              style={{
                backgroundColor: 'rgba(0, 119, 182, 0.3)',
                color: 'rgba(202, 240, 248, 0.5)',
                border: '1px solid rgba(0, 119, 182, 0.2)',
              }}
            >
              CONNECT (COMING SOON)
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
