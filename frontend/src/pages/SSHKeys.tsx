import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import FluidBackground from '@/components/effects/FluidBackground'

interface SSHKey {
  id: string
  name: string
  public_key: string
  fingerprint: string
  is_default: boolean
  created_at: string
}

export default function SSHKeys() {
  const [keys, setKeys] = useState<SSHKey[]>([])
  const [loading, setLoading] = useState(true)
  const [showAdd, setShowAdd] = useState(false)
  const [name, setName] = useState('')
  const [publicKey, setPublicKey] = useState('')
  const [message, setMessage] = useState('')
  const navigate = useNavigate()

  const fetchKeys = useCallback(() => {
    const token = localStorage.getItem('access_token')
    if (!token) { navigate('/login'); return }
    fetch('/v1/ssh-keys', { headers: { Authorization: `Bearer ${token}` } })
      .then(r => r.ok ? r.json() : [])
      .then(data => { setKeys(Array.isArray(data) ? data : []) })
      .catch(() => setKeys([]))
      .finally(() => setLoading(false))
  }, [navigate])

  useEffect(() => { fetchKeys() }, [fetchKeys])
  useEffect(() => { if (!localStorage.getItem('access_token')) navigate('/login') }, [navigate])

  const addKey = async () => {
    if (!name || !publicKey) { setMessage('Fill in all fields'); return }
    const token = localStorage.getItem('access_token')
    const resp = await fetch('/v1/ssh-keys', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ name, public_key: publicKey }),
    })
    if (resp.ok) {
      setShowAdd(false)
      setName('')
      setPublicKey('')
      setMessage('')
      fetchKeys()
    } else {
      const data = await resp.json()
      setMessage(data.error || 'Failed to add key')
    }
  }

  const deleteKey = async (id: string) => {
    const token = localStorage.getItem('access_token')
    await fetch(`/v1/ssh-keys/${id}`, { method: 'DELETE', headers: { Authorization: `Bearer ${token}` } })
    fetchKeys()
  }

  return (
    <div className="relative min-h-screen" style={{ backgroundColor: 'var(--abyssal-black)' }}>
      <FluidBackground />
      <nav className="fixed top-0 left-0 right-0 z-50" style={{ backdropFilter: 'blur(20px)', borderBottom: '1px solid rgba(0, 119, 182, 0.15)', backgroundColor: 'rgba(0, 13, 29, 0.85)' }}>
        <div className="mx-auto flex max-w-[1400px] items-center justify-between px-6 py-4 md:px-12">
          <a href="/dashboard" className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded-full" style={{ border: '1px solid rgba(144, 224, 239, 0.4)' }}>
              <div className="h-2 w-2 rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
            </div>
            <span className="font-mono text-sm tracking-[0.15em]" style={{ color: 'var(--surface-mist)' }}>AETHERIUS</span>
          </a>
          <a href="/dashboard" className="rounded-full px-4 py-1.5 font-mono text-xs tracking-wider" style={{ border: '1px solid rgba(0, 119, 182, 0.3)', color: 'rgba(202, 240, 248, 0.7)' }}>
            Dashboard
          </a>
        </div>
      </nav>

      <div className="relative px-6 pt-32 pb-20 md:px-12">
        <div className="mx-auto max-w-[800px]">
          <div className="mb-8 flex items-center justify-between">
            <div>
              <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>SSH KEYS</p>
              <h1 className="mt-2 font-sans text-[clamp(1.5rem,3vw,2.5rem)] font-semibold leading-[1.1] tracking-[-0.02em]" style={{ color: 'var(--glacial-cyan)' }}>
                SSH Key Management
              </h1>
              <p className="mt-2 font-mono text-xs" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>
                Add your public SSH key to access deployed containers via SSH.
              </p>
            </div>
            <button onClick={() => setShowAdd(true)} className="rounded-lg px-4 py-2 font-mono text-xs tracking-[0.15em]" style={{ backgroundColor: 'var(--core-blue)', color: 'var(--surface-mist)' }}>
              + ADD KEY
            </button>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-20">
              <span className="h-2 w-2 animate-pulse rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
            </div>
          ) : keys.length === 0 && !showAdd ? (
            <div className="rounded-xl p-12 text-center" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
              <p className="font-mono text-sm tracking-[0.15em]" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>NO SSH KEYS</p>
              <p className="mt-2 font-mono text-xs" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>Add a public key to SSH into your GPU deployments.</p>
            </div>
          ) : (
            <div className="space-y-3">
              {showAdd && (
                <div className="rounded-xl p-5" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
                  <div className="space-y-3">
                    <div>
                      <p className="mb-1 font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>KEY NAME</p>
                      <input value={name} onChange={e => setName(e.target.value)} placeholder="e.g. My MacBook Pro" className="w-full rounded-lg px-3 py-2 font-mono text-xs outline-none" style={{ backgroundColor: 'rgba(0, 8, 20, 0.6)', border: '1px solid rgba(0, 119, 182, 0.15)', color: 'var(--surface-mist)' }} />
                    </div>
                    <div>
                      <p className="mb-1 font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>PUBLIC KEY</p>
                      <textarea value={publicKey} onChange={e => setPublicKey(e.target.value)} placeholder="ssh-rsa AAAAB3..." rows={3} className="w-full rounded-lg px-3 py-2 font-mono text-xs outline-none resize-none" style={{ backgroundColor: 'rgba(0, 8, 20, 0.6)', border: '1px solid rgba(0, 119, 182, 0.15)', color: 'var(--surface-mist)' }} />
                    </div>
                    {message && <p className="font-mono text-xs" style={{ color: 'rgba(255, 107, 107, 0.8)' }}>{message}</p>}
                    <div className="flex gap-2">
                      <button onClick={addKey} className="rounded-lg px-4 py-2 font-mono text-xs tracking-[0.15em]" style={{ backgroundColor: 'var(--core-blue)', color: 'var(--surface-mist)' }}>SAVE</button>
                      <button onClick={() => setShowAdd(false)} className="rounded-lg px-4 py-2 font-mono text-xs tracking-[0.15em]" style={{ border: '1px solid rgba(202, 240, 248, 0.2)', color: 'rgba(202, 240, 248, 0.5)' }}>CANCEL</button>
                    </div>
                  </div>
                </div>
              )}

              {keys.map(k => (
                <div key={k.id} className="rounded-xl p-5" style={{ border: '1px solid rgba(0, 119, 182, 0.12)', backdropFilter: 'blur(8px)', background: 'rgba(0, 13, 29, 0.5)' }}>
                  <div className="flex items-start justify-between">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <p className="font-mono text-sm" style={{ color: 'var(--surface-mist)' }}>{k.name}</p>
                        {k.is_default && (
                          <span className="rounded-full px-2 py-0.5 font-mono text-[10px] tracking-wider" style={{ backgroundColor: 'rgba(34, 197, 94, 0.1)', color: 'rgba(34, 197, 94, 0.9)' }}>DEFAULT</span>
                        )}
                      </div>
                      <p className="mt-1.5 font-mono text-[10px] truncate" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>{k.public_key}</p>
                      <p className="mt-0.5 font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>Fingerprint: {k.fingerprint}</p>
                    </div>
                    <button onClick={() => deleteKey(k.id)} className="rounded-md px-3 py-1.5 font-mono text-[10px] tracking-wider" style={{ backgroundColor: 'rgba(255, 107, 107, 0.15)', color: 'rgba(255, 107, 107, 0.9)', border: '1px solid rgba(255, 107, 107, 0.2)' }}>
                      DELETE
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}