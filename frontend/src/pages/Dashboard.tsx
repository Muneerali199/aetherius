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

interface NodeInfo {
  id: string
  status: string
  total_gpu: number
  available_gpu: number
  total_vram_gb: number
  total_ram_gb: number
  total_disk_gb: number
  cpu_model: string
  cpu_cores: number
  gpu_models: string[]
  os_name: string
  region: string
  first_seen: string
  last_heartbeat: string
  created_at: string
}

type Tab = 'overview' | 'nodes' | 'connect'

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, { bg: string; text: string; dot: string }> = {
    active: { bg: 'rgba(34, 197, 94, 0.1)', text: 'rgba(34, 197, 94, 0.9)', dot: '#22c55e' },
    offline: { bg: 'rgba(107, 114, 128, 0.1)', text: 'rgba(107, 114, 128, 0.7)', dot: '#6b7280' },
    pending: { bg: 'rgba(234, 179, 8, 0.1)', text: 'rgba(234, 179, 8, 0.9)', dot: '#eab308' },
    paused: { bg: 'rgba(59, 130, 246, 0.1)', text: 'rgba(59, 130, 246, 0.9)', dot: '#3b82f6' },
  }
  const c = colors[status] || colors.offline

  return (
    <span className="inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 font-mono text-[10px] tracking-wider" style={{ backgroundColor: c.bg, color: c.text }}>
      <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: c.dot }} />
      {status.toUpperCase()}
    </span>
  )
}

export default function Dashboard() {
  const [user, setUser] = useState<UserData | null>(null)
  const [nodes, setNodes] = useState<NodeInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [tab, setTab] = useState<Tab>('overview')
  const navigate = useNavigate()

  useEffect(() => {
    const token = localStorage.getItem('access_token')
    if (!token) { navigate('/login'); return }

    const headers = { Authorization: `Bearer ${token}` }

    Promise.all([
      fetch('/v1/auth/me', { headers }).then(r => r.ok ? r.json() : Promise.reject()),
      fetch('/v1/nodes', { headers }).then(r => r.ok ? r.json() : []),
    ])
      .then(([userData, nodesData]) => {
        setUser(userData)
        setNodes(nodesData)
      })
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

  const activeNodes = nodes.filter(n => n.status === 'active')
  const totalGPU = nodes.reduce((s, n) => s + n.total_gpu, 0)
  const availableGPU = nodes.reduce((s, n) => s + n.available_gpu, 0)
  const totalRAM = nodes.reduce((s, n) => s + n.total_ram_gb, 0)
  const totalVRAM = nodes.reduce((s, n) => s + n.total_vram_gb, 0)

  const tabs: { key: Tab; label: string }[] = [
    { key: 'overview', label: 'OVERVIEW' },
    { key: 'nodes', label: `NODES (${nodes.length})` },
    { key: 'connect', label: 'CONNECT' },
  ]

  return (
    <div className="relative min-h-screen" style={{ backgroundColor: 'var(--abyssal-black)' }}>
      <FluidBackground />

      <nav className="fixed top-0 left-0 right-0 z-50" style={{ backdropFilter: 'blur(20px)', borderBottom: '1px solid rgba(0, 119, 182, 0.15)', backgroundColor: 'rgba(0, 13, 29, 0.85)' }}>
        <div className="mx-auto flex max-w-[1400px] items-center justify-between px-6 py-4 md:px-12">
          <a href="/" className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-full" style={{ border: '1px solid rgba(144, 224, 239, 0.4)' }}>
              <div className="h-2 w-2 rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
            </div>
            <span className="font-mono text-sm tracking-[0.15em]" style={{ color: 'var(--surface-mist)' }}>AETHERIUS</span>
          </a>
          <div className="flex items-center gap-4">
            <span className="hidden font-mono text-xs tracking-[0.1em] md:block" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>{user.email}</span>
            <button onClick={handleLogout} className="rounded-full px-4 py-1.5 font-mono text-xs tracking-wider transition-all duration-300 hover:brightness-110" style={{ border: '1px solid rgba(255, 107, 107, 0.3)', color: 'rgba(255, 107, 107, 0.8)' }}>
              Disconnect
            </button>
          </div>
        </div>
      </nav>

      <div className="relative px-6 pt-32 pb-20 md:px-12">
        <div className="mx-auto max-w-[1200px]">
          <div className="mb-8">
            <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>NODE DASHBOARD</p>
            <h1 className="mt-2 font-sans text-[clamp(1.5rem,3vw,2.5rem)] font-semibold leading-[1.1] tracking-[-0.02em]" style={{ color: 'var(--glacial-cyan)' }}>
              Welcome, {user.name}
            </h1>
          </div>

          <div className="mb-8 flex gap-1 rounded-lg p-1" style={{ backgroundColor: 'rgba(0, 119, 182, 0.08)' }}>
            {tabs.map(t => (
              <button
                key={t.key}
                onClick={() => setTab(t.key)}
                className="flex-1 rounded-md px-4 py-2 font-mono text-xs tracking-[0.15em] transition-all duration-300"
                style={{
                  color: tab === t.key ? 'var(--glacial-cyan)' : 'rgba(202, 240, 248, 0.4)',
                  backgroundColor: tab === t.key ? 'rgba(0, 119, 182, 0.2)' : 'transparent',
                }}
              >
                {t.label}
              </button>
            ))}
          </div>

          {tab === 'overview' && (
            <div className="space-y-6">
              <div className="grid gap-4 md:grid-cols-4">
                {[
                  { label: 'Total Nodes', value: nodes.length.toString(), sub: `${activeNodes.length} active` },
                  { label: 'GPU Capacity', value: totalGPU.toString(), sub: `${availableGPU} available` },
                  { label: 'VRAM', value: `${totalVRAM} GB`, sub: 'total capacity' },
                  { label: 'RAM', value: `${totalRAM} GB`, sub: 'total capacity' },
                ].map(s => (
                  <div key={s.label} className="rounded-xl p-5" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
                    <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>{s.label}</p>
                    <p className="mt-2 font-sans text-2xl font-semibold tracking-tight" style={{ color: 'var(--glacial-cyan)' }}>{s.value}</p>
                    <p className="mt-1 font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>{s.sub}</p>
                  </div>
                ))}
              </div>

              <div className="rounded-xl p-6" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
                <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>RECENT NODES</p>
                {nodes.length === 0 ? (
                  <div className="mt-6 text-center">
                    <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full" style={{ border: '1px solid rgba(144, 224, 239, 0.15)' }}>
                      <div className="h-2 w-2 rounded-full" style={{ backgroundColor: 'rgba(144, 224, 239, 0.3)' }} />
                    </div>
                    <p className="mt-3 font-mono text-xs tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>
                      No nodes connected yet
                    </p>
                    <Button onClick={() => setTab('connect')} className="mt-4 h-9 font-mono text-xs tracking-[0.15em]" style={{ backgroundColor: 'var(--core-blue)', color: 'var(--surface-mist)' }}>
                      CONNECT A NODE
                    </Button>
                  </div>
                ) : (
                  <div className="mt-4 space-y-2">
                    {nodes.slice(0, 5).map(n => (
                      <div key={n.id} className="flex items-center justify-between rounded-lg px-4 py-3" style={{ border: '1px solid rgba(0, 119, 182, 0.1)' }}>
                        <div>
                          <p className="font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>{n.cpu_model || 'Unknown'} ({n.cpu_cores} cores)</p>
                          <p className="mt-0.5 font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>{n.os_name} · {n.region}</p>
                        </div>
                        <StatusBadge status={n.status} />
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {tab === 'nodes' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>ALL NODES</p>
                <Button onClick={() => setTab('connect')} className="h-8 font-mono text-xs tracking-[0.15em]" style={{ backgroundColor: 'var(--core-blue)', color: 'var(--surface-mist)' }}>
                  + CONNECT NODE
                </Button>
              </div>

              {nodes.length === 0 ? (
                <div className="rounded-xl p-12 text-center" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
                  <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full" style={{ border: '1px solid rgba(144, 224, 239, 0.15)' }}>
                    <div className="h-3 w-3 rounded-full" style={{ backgroundColor: 'rgba(144, 224, 239, 0.3)' }} />
                  </div>
                  <p className="mt-4 font-mono text-sm tracking-[0.15em]" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>NO NODES REGISTERED</p>
                  <p className="mx-auto mt-2 max-w-md text-sm leading-relaxed" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>
                    Deploy your first compute node to start earning. Download the agent and run the one-line install command.
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  {nodes.map(n => (
                    <div key={n.id} className="rounded-xl p-5" style={{ border: '1px solid rgba(0, 119, 182, 0.12)', backdropFilter: 'blur(8px)', background: 'rgba(0, 13, 29, 0.5)' }}>
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-3">
                            <p className="font-mono text-sm" style={{ color: 'var(--surface-mist)' }}>{n.cpu_model || 'Unknown CPU'}</p>
                            <StatusBadge status={n.status} />
                          </div>
                          <div className="mt-3 grid grid-cols-2 gap-4 md:grid-cols-4">
                            <div>
                              <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>GPU</p>
                              <p className="mt-0.5 font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>{n.available_gpu}/{n.total_gpu}</p>
                            </div>
                            <div>
                              <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>VRAM</p>
                              <p className="mt-0.5 font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>{n.total_vram_gb} GB</p>
                            </div>
                            <div>
                              <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>RAM</p>
                              <p className="mt-0.5 font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>{n.total_ram_gb} GB</p>
                            </div>
                            <div>
                              <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>REGION</p>
                              <p className="mt-0.5 font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>{n.region || '—'}</p>
                            </div>
                          </div>
                          {n.gpu_models.length > 0 && (
                            <div className="mt-3 flex flex-wrap gap-1.5">
                              {n.gpu_models.map((gpu, i) => (
                                <span key={i} className="rounded-md px-2 py-0.5 font-mono text-[10px]" style={{ backgroundColor: 'rgba(0, 119, 182, 0.1)', color: 'rgba(144, 224, 239, 0.7)', border: '1px solid rgba(0, 119, 182, 0.15)' }}>
                                  {gpu}
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {tab === 'connect' && (
            <div className="mx-auto max-w-2xl text-center">
              <div className="rounded-xl p-8" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
                <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full" style={{ border: '1px solid rgba(144, 224, 239, 0.2)' }}>
                  <div className="h-3 w-3 rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
                </div>
                <h2 className="mt-4 font-mono text-sm tracking-[0.15em]" style={{ color: 'var(--glacial-cyan)' }}>
                  DEPLOY A COMPUTE NODE
                </h2>
                <p className="mx-auto mt-2 max-w-md text-sm leading-relaxed" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>
                  Run this one-line command on any Linux, macOS, or Windows machine with an NVIDIA GPU:
                </p>
                <div className="mt-6 rounded-lg p-4 text-left" style={{ backgroundColor: 'rgba(0, 8, 20, 0.8)', border: '1px solid rgba(0, 119, 182, 0.15)' }}>
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(144, 224, 239, 0.5)' }}>INSTALL.SH</span>
                    <button
                      onClick={() => navigator.clipboard.writeText(`curl -fsSL https://get.aetherius.io | sh`)}
                      className="rounded px-2 py-0.5 font-mono text-[10px] tracking-wider transition-all duration-200"
                      style={{ color: 'rgba(144, 224, 239, 0.6)', backgroundColor: 'rgba(0, 119, 182, 0.15)' }}
                    >
                      COPY
                    </button>
                  </div>
                  <pre className="mt-3 overflow-x-auto font-mono text-sm leading-relaxed" style={{ color: 'var(--glacial-cyan)' }}>
                    <code>{`curl -fsSL https://get.aetherius.io | sh`}</code>
                  </pre>
                </div>
                <div className="mt-6 space-y-3 text-left">
                  {[
                    'Downloads and verifies the Aetherius node agent',
                    'Auto-detects your GPU, CPU, RAM, and disk specs',
                    'Registers your node with the network',
                    'Starts earning rewards immediately based on available resources',
                  ].map((step, i) => (
                    <div key={i} className="flex items-start gap-3">
                      <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full font-mono text-[10px]" style={{ backgroundColor: 'rgba(0, 119, 182, 0.2)', color: 'var(--glacial-cyan)' }}>
                        {i + 1}
                      </span>
                      <span className="font-mono text-xs leading-relaxed" style={{ color: 'rgba(202, 240, 248, 0.6)' }}>{step}</span>
                    </div>
                  ))}
                </div>
                <div className="mt-6 rounded-lg p-4" style={{ backgroundColor: 'rgba(0, 119, 182, 0.06)', border: '1px solid rgba(0, 119, 182, 0.1)' }}>
                  <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(144, 224, 239, 0.4)' }}>REQUIREMENTS</p>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {['NVIDIA GPU with 8GB+ VRAM', 'Linux / macOS / Windows', 'Docker installed', 'Internet connection'].map(req => (
                      <span key={req} className="rounded-md px-2.5 py-1 font-mono text-[10px]" style={{ backgroundColor: 'rgba(0, 119, 182, 0.08)', color: 'rgba(202, 240, 248, 0.5)', border: '1px solid rgba(0, 119, 182, 0.1)' }}>
                        {req}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
