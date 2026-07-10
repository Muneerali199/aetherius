import { useEffect, useMemo, useState } from 'react'
import FluidBackground from '@/components/effects/FluidBackground'

interface AvailableNode {
  id: string
  total_gpu: number
  available_gpu: number
  total_vram_gb: number
  available_vram_gb: number
  total_ram_gb: number
  total_disk_gb: number
  gpu_models: string[]
  region: string
  estimated_price_per_hour: number
  status: string
  last_heartbeat: string
}

function GPUModelTag({ model }: { model: string }) {
  const isNVIDIA = model.toLowerCase().includes('nvidia') || model.includes('RTX') || model.includes('GTX') || model.includes('Tesla') || model.includes('A100') || model.includes('H100')
  const isAMD = model.toLowerCase().includes('amd') || model.includes('Radeon')
  const color = isNVIDIA ? '#76b900' : isAMD ? '#e02b20' : 'rgba(144, 224, 239, 0.7)'
  return (
    <span className="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1 font-mono text-[10px]" style={{ backgroundColor: `${color}15`, color, border: `1px solid ${color}30` }}>
      {isNVIDIA && <span className="text-[8px]">◆</span>}
      {isAMD && <span className="text-[8px]">▲</span>}
      {model}
    </span>
  )
}

function PriceDisplay({ price }: { price: number }) {
  return (
    <span className="font-mono" style={{ color: 'var(--glacial-cyan)' }}>
      ${price.toFixed(3)}<span className="text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>/hr</span>
    </span>
  )
}

export default function Marketplace() {
  const [nodes, setNodes] = useState<AvailableNode[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [minGPU, setMinGPU] = useState(0)
  const [maxPrice, setMaxPrice] = useState(10)
  const [showDeployModal, setShowDeployModal] = useState<AvailableNode | null>(null)

  useEffect(() => {
    fetch('/v1/nodes/available')
      .then(r => r.ok ? r.json() : [])
      .then(data => {
        const arr = Array.isArray(data) ? data : []
        setNodes(arr)
      })
      .catch(() => setNodes([]))
      .finally(() => setLoading(false))
  }, [])

  const filtered = useMemo(() => {
    return nodes.filter(n => {
      if (n.available_gpu < minGPU) return false
      if (n.estimated_price_per_hour > maxPrice) return false
      if (search) {
        const q = search.toLowerCase()
        const matchesGPU = n.gpu_models.some(g => g.toLowerCase().includes(q))
        const matchesRegion = n.region.toLowerCase().includes(q)
        if (!matchesGPU && !matchesRegion) return false
      }
      return true
    })
  }, [nodes, search, minGPU, maxPrice])

  const totalGPUCapacity = nodes.reduce((s, n) => s + n.total_gpu, 0)
  const availableGPU = nodes.reduce((s, n) => s + n.available_gpu, 0)
  const cheapestPrice = nodes.length ? Math.min(...nodes.map(n => n.estimated_price_per_hour)) : 0

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
            <a href="/dashboard" className="rounded-full px-4 py-1.5 font-mono text-xs tracking-wider transition-all duration-300" style={{ border: '1px solid rgba(0, 119, 182, 0.3)', color: 'rgba(202, 240, 248, 0.7)' }}>
              Dashboard
            </a>
            <a href="/login" className="rounded-full px-4 py-1.5 font-mono text-xs tracking-wider transition-all duration-300 hover:brightness-110" style={{ border: '1px solid rgba(0, 119, 182, 0.4)', color: 'var(--glacial-cyan)' }}>
              Sign In
            </a>
          </div>
        </div>
      </nav>

      <div className="relative px-6 pt-32 pb-20 md:px-12">
        <div className="mx-auto max-w-[1200px]">
          <div className="mb-8">
            <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>GPU MARKETPLACE</p>
            <h1 className="mt-2 font-sans text-[clamp(1.5rem,3vw,2.5rem)] font-semibold leading-[1.1] tracking-[-0.02em]" style={{ color: 'var(--glacial-cyan)' }}>
              Rent GPU Compute
            </h1>
            <p className="mt-2 max-w-lg font-mono text-xs leading-relaxed" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>
              Browse available GPU nodes across the Aetherius network. Rent by the hour for AI training, rendering, or general compute.
            </p>
          </div>

          <div className="mb-8 grid gap-4 md:grid-cols-4">
            {[
              { label: 'Available GPUs', value: `${availableGPU}/${totalGPUCapacity}`, sub: 'across network' },
              { label: 'Active Nodes', value: nodes.length.toString(), sub: 'online now' },
              { label: 'Cheapest GPU', value: `$${cheapestPrice.toFixed(3)}/hr`, sub: 'starting price' },
              { label: 'GPU Models', value: new Set(nodes.flatMap(n => n.gpu_models)).size.toString(), sub: 'unique models' },
            ].map(s => (
              <div key={s.label} className="rounded-xl p-5" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
                <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>{s.label}</p>
                <p className="mt-2 font-sans text-2xl font-semibold tracking-tight" style={{ color: 'var(--glacial-cyan)' }}>{s.value}</p>
                <p className="mt-1 font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>{s.sub}</p>
              </div>
            ))}
          </div>

          <div className="mb-6 flex flex-wrap gap-3 rounded-xl p-4" style={{ border: '1px solid rgba(0, 119, 182, 0.12)', backdropFilter: 'blur(8px)', background: 'rgba(0, 13, 29, 0.5)' }}>
            <input
              placeholder="Search GPU model or region..."
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="flex-1 min-w-[200px] rounded-lg px-3 py-2 font-mono text-xs tracking-wider outline-none"
              style={{ backgroundColor: 'rgba(0, 8, 20, 0.6)', border: '1px solid rgba(0, 119, 182, 0.15)', color: 'var(--surface-mist)' }}
            />
            <select
              value={minGPU}
              onChange={e => setMinGPU(Number(e.target.value))}
              className="rounded-lg px-3 py-2 font-mono text-xs tracking-wider outline-none"
              style={{ backgroundColor: 'rgba(0, 8, 20, 0.6)', border: '1px solid rgba(0, 119, 182, 0.15)', color: 'rgba(202, 240, 248, 0.7)' }}
            >
              <option value={0}>Min GPUs: Any</option>
              <option value={1}>1+ GPU</option>
              <option value={2}>2+ GPUs</option>
              <option value={4}>4+ GPUs</option>
              <option value={8}>8+ GPUs</option>
            </select>
            <input
              type="range"
              min={0}
              max={10}
              step={0.5}
              value={maxPrice}
              onChange={e => setMaxPrice(Number(e.target.value))}
              className="w-32"
            />
            <span className="font-mono text-[10px] tracking-wider self-center" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>
              Max ${maxPrice.toFixed(1)}/hr
            </span>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-20">
              <span className="h-2 w-2 animate-pulse rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
              <span className="ml-3 font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>SCANNING NETWORK...</span>
            </div>
          ) : filtered.length === 0 ? (
            <div className="rounded-xl p-12 text-center" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
              <div className="mx-auto flex h-16 w-16 items-center justify-center rounded-full" style={{ border: '1px solid rgba(144, 224, 239, 0.15)' }}>
                <div className="h-3 w-3 rounded-full" style={{ backgroundColor: 'rgba(144, 224, 239, 0.3)' }} />
              </div>
              <p className="mt-4 font-mono text-sm tracking-[0.15em]" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>NO GPU NODES AVAILABLE</p>
              <p className="mx-auto mt-2 max-w-md text-sm leading-relaxed" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>
                No nodes match your filters. Try adjusting your search criteria or check back later.
              </p>
            </div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2">
              {filtered.map(node => (
                <div key={node.id} className="rounded-xl p-5 transition-all duration-300 hover:translate-y-[-2px]" style={{ border: '1px solid rgba(0, 119, 182, 0.12)', backdropFilter: 'blur(8px)', background: 'rgba(0, 13, 29, 0.5)' }}>
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 font-mono text-[10px] tracking-wider" style={{ backgroundColor: node.available_gpu > 0 ? 'rgba(34, 197, 94, 0.1)' : 'rgba(107, 114, 128, 0.1)', color: node.available_gpu > 0 ? 'rgba(34, 197, 94, 0.9)' : 'rgba(107, 114, 128, 0.7)' }}>
                          <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: node.available_gpu > 0 ? '#22c55e' : '#6b7280' }} />
                          {node.available_gpu > 0 ? `${node.available_gpu} GPU${node.available_gpu > 1 ? 's' : ''} available` : 'Fully booked'}
                        </span>
                        <PriceDisplay price={node.estimated_price_per_hour} />
                      </div>

                      {node.gpu_models.length > 0 && (
                        <div className="mt-3 flex flex-wrap gap-1.5">
                          {node.gpu_models.map((gpu, i) => (
                            <GPUModelTag key={i} model={gpu} />
                          ))}
                        </div>
                      )}

                      <div className="mt-3 grid grid-cols-2 gap-3 md:grid-cols-4">
                        <div>
                          <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>VRAM</p>
                          <p className="mt-0.5 font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>{node.total_vram_gb} GB</p>
                        </div>
                        <div>
                          <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>RAM</p>
                          <p className="mt-0.5 font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>{node.total_ram_gb} GB</p>
                        </div>
                        <div>
                          <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>DISK</p>
                          <p className="mt-0.5 font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>{node.total_disk_gb} GB</p>
                        </div>
                        <div>
                          <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>REGION</p>
                          <p className="mt-0.5 font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>{node.region || '—'}</p>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="mt-4 flex gap-2">
                    <button
                      onClick={() => setShowDeployModal(node)}
                      disabled={node.available_gpu === 0}
                      className="flex-1 rounded-lg px-4 py-2 font-mono text-xs tracking-[0.15em] transition-all duration-300 hover:brightness-110 disabled:opacity-30"
                      style={{ backgroundColor: 'var(--core-blue)', color: 'var(--surface-mist)' }}
                    >
                      RENT GPU
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {showDeployModal && (
        <DeployModal
          node={showDeployModal}
          onClose={() => setShowDeployModal(null)}
        />
      )}
    </div>
  )
}

function DeployModal({ node, onClose }: { node: AvailableNode; onClose: () => void }) {
  const [image, setImage] = useState('nvidia/cuda:12.2-runtime-ubuntu22.04')
  const [gpuCount, setGpuCount] = useState(1)
  const [deploying, setDeploying] = useState(false)
  const [result, setResult] = useState<string | null>(null)

  const handleDeploy = async () => {
    const token = localStorage.getItem('access_token')
    if (!token) { window.location.href = '/login'; return }

    setDeploying(true)
    setResult(null)

    try {
      const resp = await fetch('/v1/deployments', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          image,
          gpu_required: gpuCount,
          vram_required_gb: Math.round(node.total_vram_gb / node.total_gpu * gpuCount),
          ram_required_gb: 4,
          disk_required_gb: 20,
          ports: {},
          env: {},
        }),
      })
      if (resp.status === 401) {
        localStorage.removeItem('access_token')
        localStorage.removeItem('refresh_token')
        window.location.href = '/login'
        return
      }
      const data = await resp.json()
      if (resp.ok) {
        setResult(`Deployment created! ID: ${data.id.slice(0, 12)}...`)
        setTimeout(() => window.location.href = '/dashboard#deployments', 2000)
      } else {
        setResult(data.error || 'Deployment failed')
      }
    } catch {
      setResult('Network error')
    } finally {
      setDeploying(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ backgroundColor: 'rgba(0, 0, 0, 0.7)', backdropFilter: 'blur(4px)' }}>
      <div className="w-full max-w-md rounded-xl p-6" style={{ border: '1px solid rgba(0, 119, 182, 0.2)', backgroundColor: 'rgba(0, 13, 29, 0.95)' }}>
        <div className="flex items-center justify-between mb-4">
          <h2 className="font-mono text-sm tracking-[0.15em]" style={{ color: 'var(--glacial-cyan)' }}>DEPLOY ON GPU NODE</h2>
          <button onClick={onClose} className="font-mono text-xs" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>✕</button>
        </div>

        <div className="mb-4 rounded-lg p-3" style={{ backgroundColor: 'rgba(0, 119, 182, 0.06)', border: '1px solid rgba(0, 119, 182, 0.1)' }}>
          <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(144, 224, 239, 0.5)' }}>NODE SPECS</p>
          <p className="mt-1 font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>
            {node.gpu_models.join(', ')} · {node.total_vram_gb}GB VRAM · {node.total_ram_gb}GB RAM
          </p>
        </div>

        <div className="space-y-4">
          <div>
            <p className="mb-1 font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>DOCKER IMAGE</p>
            <input
              value={image}
              onChange={e => setImage(e.target.value)}
              className="w-full rounded-lg px-3 py-2 font-mono text-xs outline-none"
              style={{ backgroundColor: 'rgba(0, 8, 20, 0.6)', border: '1px solid rgba(0, 119, 182, 0.15)', color: 'var(--surface-mist)' }}
            />
          </div>
          <div>
            <p className="mb-1 font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>GPUs TO USE</p>
            <select
              value={gpuCount}
              onChange={e => setGpuCount(Number(e.target.value))}
              className="w-full rounded-lg px-3 py-2 font-mono text-xs outline-none"
              style={{ backgroundColor: 'rgba(0, 8, 20, 0.6)', border: '1px solid rgba(0, 119, 182, 0.15)', color: 'rgba(202, 240, 248, 0.7)' }}
            >
              {Array.from({ length: node.available_gpu }, (_, i) => i + 1).map(n => (
                <option key={n} value={n}>{n} GPU{n > 1 ? 's' : ''}</option>
              ))}
            </select>
          </div>

          <div className="rounded-lg p-3" style={{ backgroundColor: 'rgba(0, 119, 182, 0.06)', border: '1px solid rgba(0, 119, 182, 0.1)' }}>
            <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(144, 224, 239, 0.5)' }}>ESTIMATED COST</p>
            <p className="mt-1 font-sans text-lg font-semibold tracking-tight" style={{ color: 'var(--glacial-cyan)' }}>
              ${(node.estimated_price_per_hour / node.total_gpu * gpuCount).toFixed(3)}<span className="text-xs" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>/hr</span>
            </p>
          </div>

          {result && (
            <div className="rounded-lg p-3 font-mono text-xs" style={{ backgroundColor: result.includes('Deployment created') ? 'rgba(34, 197, 94, 0.1)' : 'rgba(255, 107, 107, 0.1)', border: `1px solid ${result.includes('Deployment created') ? 'rgba(34, 197, 94, 0.2)' : 'rgba(255, 107, 107, 0.2)'}`, color: result.includes('Deployment created') ? 'rgba(34, 197, 94, 0.9)' : 'rgba(255, 107, 107, 0.8)' }}>
              {result}
            </div>
          )}

          <button
            onClick={handleDeploy}
            disabled={deploying}
            className="w-full rounded-lg px-4 py-2.5 font-mono text-xs tracking-[0.15em] transition-all duration-300 hover:brightness-110 disabled:opacity-50"
            style={{ backgroundColor: 'var(--core-blue)', color: 'var(--surface-mist)' }}
          >
            {deploying ? 'DEPLOYING...' : 'START DEPLOYMENT'}
          </button>
        </div>
      </div>
    </div>
  )
}