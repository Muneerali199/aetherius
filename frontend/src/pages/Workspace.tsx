import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

interface DeploymentInfo {
  id: string
  user_id: string
  node_id: string | null
  image: string
  gpu_required: number
  vram_required_gb: number
  ram_required_gb: number
  disk_required_gb: number
  status: string
  cost_per_hour: number
  region: string
  created_at: string
  updated_at: string
}

const statusColors: Record<string, string> = {
  pending: '#eab308',
  scheduling: '#3b82f6',
  pulling: '#a855f7',
  running: '#22c55e',
  stopped: '#6b7280',
  failed: '#ff6b6b',
}

interface FileNode {
  name: string
  type: 'file' | 'folder'
  children?: FileNode[]
}

function FileTree() {
  const files: FileNode[] = [
    { name: 'workspace', type: 'folder', children: [
      { name: 'train.py', type: 'file' },
      { name: 'model.py', type: 'file' },
      { name: 'requirements.txt', type: 'file' },
      { name: 'data', type: 'folder', children: [
        { name: 'sample.csv', type: 'file' },
      ]},
    ]},
    { name: '.ssh', type: 'folder', children: [
      { name: 'authorized_keys', type: 'file' },
    ]},
  ]

  return (
    <div className="py-2">
      <p className="px-3 pb-1 font-mono text-[10px] tracking-wider uppercase" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>EXPLORER</p>
      <div className="space-y-0.5">
        <TreeNode name="~" type="folder" defaultOpen children={files} />
      </div>
    </div>
  )
}

function TreeNode({ name, type, children, defaultOpen }: { name: string; type: 'file' | 'folder'; children?: FileNode[]; defaultOpen?: boolean }) {
  const [open, setOpen] = useState(defaultOpen || false)
  return (
    <div>
      <button
        onClick={() => type === 'folder' && setOpen(!open)}
        className="flex w-full items-center gap-1.5 px-3 py-0.5 text-left font-mono text-xs transition-colors hover:brightness-125"
        style={{ color: type === 'folder' ? 'rgba(202, 240, 248, 0.7)' : 'rgba(202, 240, 248, 0.5)' }}
      >
        {type === 'folder' && (
          <span style={{ color: 'rgba(144, 224, 239, 0.4)', fontSize: '8px' }}>{open ? '▾' : '▸'}</span>
        )}
        {type === 'folder' ? (
          <span style={{ color: open ? 'rgba(144, 224, 239, 0.7)' : 'rgba(144, 224, 239, 0.5)' }}>📁</span>
        ) : (
          <span style={{ fontSize: '10px' }}>📄</span>
        )}
        {name}
      </button>
      {type === 'folder' && open && children && (
        <div className="ml-3">{children.map((c, i) => <TreeNode key={i} {...c} />)}</div>
      )}
    </div>
  )
}

function TermPanel({ deploymentId }: { deploymentId: string }) {
  const termRef = useRef<HTMLDivElement>(null)
  const terminalRef = useRef<Terminal | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    if (!termRef.current || terminalRef.current) return

    const term = new Terminal({
      cursorBlink: true,
      cursorStyle: 'block',
      fontSize: 13,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      allowProposedApi: true,
      theme: {
        background: '#0a0f1a',
        foreground: '#caf0f8',
        cursor: '#90e0ef',
        selectionBackground: 'rgba(0, 119, 182, 0.3)',
        black: '#1a1a2e',
        red: '#ff6b6b',
        green: '#22c55e',
        yellow: '#eab308',
        blue: '#3b82f6',
        magenta: '#a855f7',
        cyan: '#22d3ee',
        white: '#caf0f8',
        brightBlack: '#4a4a6a',
        brightRed: '#ff8787',
        brightGreen: '#4ade80',
        brightYellow: '#facc15',
        brightBlue: '#60a5fa',
        brightMagenta: '#c084fc',
        brightCyan: '#67e8f9',
        brightWhite: '#f0f9ff',
      },
    })

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(termRef.current)
    fitAddon.fit()

    term.writeln('\x1b[36m╔══════════════════════════════════════════╗\x1b[0m')
    term.writeln('\x1b[36m║   \x1b[0m  AETHERIUS REMOTE TERMINAL           \x1b[36m║\x1b[0m')
    term.writeln('\x1b[36m╚══════════════════════════════════════════╝\x1b[0m')
    term.writeln('\x1b[90mConnecting to workspace container...\x1b[0m')

    terminalRef.current = term

    // Connect WebSocket with token as query param
    const token = localStorage.getItem('access_token')
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    const ws = new WebSocket(`${protocol}//${host}/v1/workspace/${deploymentId}/terminal?token=${token}`)

    ws.onopen = () => {
      term.writeln('\x1b[32mConnected to container terminal.\x1b[0m')
      term.focus()
    }

    ws.onmessage = (event) => {
      if (event.data instanceof Blob) {
        event.data.arrayBuffer().then(buf => term.write(new Uint8Array(buf)))
      } else {
        term.write(event.data)
      }
    }

    ws.onerror = () => {
      term.writeln('\x1b[31mConnection error. Terminal unavailable.\x1b[0m')
    }

    ws.onclose = () => {
      term.writeln('\x1b[33mConnection closed.\x1b[0m')
    }

    wsRef.current = ws

    // Terminal input -> WebSocket
    term.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })

    const handleResize = () => fitAddon.fit()
    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
      ws.close()
      term.dispose()
      terminalRef.current = null
      wsRef.current = null
    }
  }, [deploymentId])

  return (
    <div className="relative h-full w-full overflow-hidden" style={{ backgroundColor: '#0a0f1a' }}>
      <div className="flex items-center gap-1.5 px-4 py-1.5" style={{ backgroundColor: '#0d1520', borderBottom: '1px solid rgba(0, 119, 182, 0.1)' }}>
        <span className="inline-block h-2.5 w-2.5 rounded-full" style={{ backgroundColor: '#ff6b6b' }} />
        <span className="inline-block h-2.5 w-2.5 rounded-full" style={{ backgroundColor: '#eab308' }} />
        <span className="inline-block h-2.5 w-2.5 rounded-full" style={{ backgroundColor: '#22c55e' }} />
        <span className="ml-3 font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>TERMINAL</span>
      </div>
      <div ref={termRef} className="h-[calc(100%-32px)] w-full" />
    </div>
  )
}

export default function Workspace() {
  const { id } = useParams()
  const navigate = useNavigate()
  const [deployment, setDeployment] = useState<DeploymentInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [activePanel, setActivePanel] = useState<'terminal' | 'ssh' | 'logs'>('terminal')

  const fetchDeployment = useCallback(() => {
    const token = localStorage.getItem('access_token')
    if (!token) { navigate('/login'); return }
    fetch(`/v1/deployments/${id}`, { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.ok ? r.json() : null)
      .then(d => {
        if (!d) navigate('/dashboard')
        setDeployment(d)
      })
      .catch(() => navigate('/dashboard'))
      .finally(() => setLoading(false))
  }, [id, navigate])

  useEffect(() => {
    if (!localStorage.getItem('access_token')) navigate('/login')
    else fetchDeployment()
  }, [fetchDeployment, navigate])

  useEffect(() => {
    if (!id) return
    const token = localStorage.getItem('access_token')
    if (!token) return
    const interval = setInterval(() => {
      fetch(`/v1/deployments/${id}`, { headers: { Authorization: `Bearer ${token}` } })
        .then(r => r.ok ? r.json() : null)
        .then(d => { if (d) setDeployment(d) })
        .catch(() => {})
    }, 10000)
    return () => clearInterval(interval)
  }, [id])

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center" style={{ backgroundColor: '#030712' }}>
        <span className="h-2 w-2 animate-pulse rounded-full" style={{ backgroundColor: '#90e0ef' }} />
      </div>
    )
  }

  if (!deployment) return null

  const statusColor = statusColors[deployment.status] || '#6b7280'
  const isRunning = deployment.status === 'running'

  return (
    <div className="flex h-screen flex-col overflow-hidden" style={{ backgroundColor: '#030712' }}>
      {/* Title bar */}
      <div className="flex items-center justify-between px-4 py-2" style={{ backgroundColor: '#0a0f1a', borderBottom: '1px solid rgba(0, 119, 182, 0.1)' }}>
        <div className="flex items-center gap-3">
          <button onClick={() => navigate('/dashboard')} className="font-mono text-xs tracking-wider transition-colors" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>
            ← BACK
          </button>
          <span className="h-4 w-px" style={{ backgroundColor: 'rgba(0, 119, 182, 0.2)' }} />
          <span className="font-mono text-xs" style={{ color: 'rgba(202, 240, 248, 0.6)' }}>{deployment.image}</span>
          <span className="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 font-mono text-[10px] tracking-wider" style={{ backgroundColor: `${statusColor}15`, color: statusColor }}>
            <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: statusColor }} />
            {deployment.status.toUpperCase()}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>
            ${deployment.cost_per_hour.toFixed(4)} / hr
          </span>
        </div>
      </div>

      {/* Main content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left sidebar — File Explorer */}
        <div className="flex w-56 flex-col overflow-y-auto" style={{ backgroundColor: '#0a0f1a', borderRight: '1px solid rgba(0, 119, 182, 0.08)' }}>
          <FileTree />
        </div>

        {/* Center — Terminal */}
        <div className="flex flex-1 flex-col overflow-hidden">
          <div className="flex-1 overflow-hidden">
            {activePanel === 'terminal' && <TermPanel deploymentId={id || ''} />}
            {activePanel === 'ssh' && (
              <div className="flex h-full items-center justify-center p-8">
                <div className="max-w-lg rounded-xl p-6" style={{ border: '1px solid rgba(0, 119, 182, 0.12)', backgroundColor: 'rgba(0, 13, 29, 0.5)' }}>
                  <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(144, 224, 239, 0.5)' }}>SSH CONNECTION</p>
                  <p className="mt-4 font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>
                    Make sure you have added your SSH public key in Settings → SSH Keys, then run:
                  </p>
                  <pre className="mt-3 overflow-x-auto rounded-md p-3 font-mono text-xs" style={{ backgroundColor: 'rgba(0, 8, 20, 0.8)', color: '#90e0ef' }}>
{`ssh -o StrictHostKeyChecking=no -p <port> root@<node-ip>`}
                  </pre>
                  <p className="mt-3 font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>
                    Port mapping and IP will be available once the deployment is scheduled to a node.
                  </p>
                  <a href="/ssh-keys" className="mt-4 inline-block font-mono text-[10px] tracking-wider" style={{ color: 'rgba(144, 224, 239, 0.6)' }}>
                    MANAGE SSH KEYS →
                  </a>
                </div>
              </div>
            )}
          </div>

          {/* Right panel — Connection Info */}
          <div className="border-t px-4 py-2" style={{ backgroundColor: '#0a0f1a', borderTopColor: 'rgba(0, 119, 182, 0.08)' }}>
            <div className="flex items-center gap-4">
              <button onClick={() => setActivePanel('terminal')} className="font-mono text-[10px] tracking-wider transition-colors" style={{ color: activePanel === 'terminal' ? '#90e0ef' : 'rgba(202, 240, 248, 0.3)' }}>
                TERMINAL
              </button>
              <button onClick={() => setActivePanel('ssh')} className="font-mono text-[10px] tracking-wider transition-colors" style={{ color: activePanel === 'ssh' ? '#90e0ef' : 'rgba(202, 240, 248, 0.3)' }}>
                SSH
              </button>
              <span className="h-3 w-px" style={{ backgroundColor: 'rgba(0, 119, 182, 0.15)' }} />
              <span className="font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>
                GPU: {deployment.gpu_required} × {deployment.vram_required_gb}GB VRAM
              </span>
              <span className="font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>
                RAM: {deployment.ram_required_gb}GB
              </span>
              {deployment.region && (
                <span className="font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>
                  REGION: {deployment.region}
                </span>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Status bar */}
      <div className="flex items-center justify-between px-4 py-1" style={{ backgroundColor: '#070b14', borderTop: '1px solid rgba(0, 119, 182, 0.08)' }}>
        <div className="flex items-center gap-3">
          <span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: isRunning ? '#22c55e' : '#6b7280' }} />
          <span className="font-mono text-[10px]" style={{ color: isRunning ? 'rgba(34, 197, 94, 0.8)' : 'rgba(107, 114, 128, 0.7)' }}>
            {isRunning ? 'CONNECTED' : deployment.status.toUpperCase()}
          </span>
          <span className="font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.2)' }}>
            LN {deployment.id.slice(0, 8)}
          </span>
        </div>
        <div className="flex items-center gap-3">
          <span className="font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.2)' }}>
            {isRunning ? 'CONTAINER ACTIVE' : '—'}
          </span>
        </div>
      </div>
    </div>
  )
}