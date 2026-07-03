import { useState } from 'react'
import { useNavigate } from 'react-router'
import FluidBackground from '@/components/effects/FluidBackground'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { cn } from '@/lib/utils'

type AuthMode = 'login' | 'register'

export default function Auth() {
  const [mode, setMode] = useState<AuthMode>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      const endpoint = mode === 'login' ? '/v1/auth/login' : '/v1/auth/register'
      const body = mode === 'login'
        ? { email, password }
        : { email, password, display_name: name }

      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })

      const data = await res.json()

      if (!res.ok) {
        throw new Error(data.error || 'Something went wrong')
      }

      if (mode === 'login') {
        if (data.mfa_required) {
          setError('MFA code required (not yet supported in browser)')
          return
        }
        localStorage.setItem('access_token', data.tokens.access_token)
        localStorage.setItem('refresh_token', data.tokens.refresh_token)
        navigate('/dashboard')
      } else {
        setMode('login')
        setError('Account created! Please sign in.')
      }
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative min-h-screen" style={{ backgroundColor: 'var(--abyssal-black)' }}>
      <FluidBackground />

      <nav className="fixed top-0 left-0 right-0 z-50">
        <div className="mx-auto flex max-w-[1400px] items-center justify-between px-6 py-4 md:px-12">
          <a href="/" className="flex items-center gap-2">
            <div
              className="flex h-8 w-8 items-center justify-center rounded-full"
              style={{ border: '1px solid rgba(144, 224, 239, 0.4)' }}
            >
              <div
                className="h-2 w-2 rounded-full"
                style={{ backgroundColor: 'var(--glacial-cyan)' }}
              />
            </div>
            <span
              className="font-mono text-sm tracking-[0.15em]"
              style={{ color: 'var(--surface-mist)' }}
            >
              AETHERIUS
            </span>
          </a>
        </div>
      </nav>

      <div className="relative flex min-h-screen items-center justify-center px-4 pt-24">
        <div className="w-full max-w-md">
          <div className="mb-8 text-center">
            <h1
              className="font-sans text-[clamp(2rem,5vw,3rem)] font-semibold leading-[1.1] tracking-[-0.02em]"
              style={{ color: 'var(--glacial-cyan)' }}
            >
              {mode === 'login' ? 'Welcome Back' : 'Join the Network'}
            </h1>
            <p
              className="mt-2 font-mono text-xs tracking-[0.2em]"
              style={{ color: 'rgba(202, 240, 248, 0.5)' }}
            >
              {mode === 'login' ? 'SIGN IN TO YOUR NODE' : 'DEPLOY YOUR FIRST NODE'}
            </p>
          </div>

          <div
            className="rounded-xl p-8"
            style={{
              border: '1px solid rgba(0, 119, 182, 0.15)',
              backdropFilter: 'blur(12px)',
              background: 'rgba(0, 13, 29, 0.6)',
            }}
          >
            <div className="mb-6 flex gap-1 rounded-lg p-1" style={{ backgroundColor: 'rgba(0, 119, 182, 0.08)' }}>
              {(['login', 'register'] as const).map((m) => (
                <button
                  key={m}
                  onClick={() => { setMode(m); setError('') }}
                  className={cn(
                    'flex-1 rounded-md px-4 py-2 font-mono text-xs tracking-[0.15em] transition-all duration-300',
                  )}
                  style={{
                    color: mode === m ? 'var(--glacial-cyan)' : 'rgba(202, 240, 248, 0.4)',
                    backgroundColor: mode === m ? 'rgba(0, 119, 182, 0.2)' : 'transparent',
                  }}
                >
                  {m === 'login' ? 'SIGN IN' : 'SIGN UP'}
                </button>
              ))}
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              {mode === 'register' && (
                <div>
                  <label
                    className="mb-1.5 block font-mono text-xs tracking-[0.15em]"
                    style={{ color: 'rgba(202, 240, 248, 0.6)' }}
                  >
                    DISPLAY NAME
                  </label>
                  <Input
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="Your node alias"
                    required
                    className="h-10"
                    style={{
                      border: '1px solid rgba(0, 119, 182, 0.2)',
                      backgroundColor: 'rgba(0, 8, 20, 0.6)',
                      color: 'var(--surface-mist)',
                    }}
                  />
                </div>
              )}

              <div>
                <label
                  className="mb-1.5 block font-mono text-xs tracking-[0.15em]"
                  style={{ color: 'rgba(202, 240, 248, 0.6)' }}
                >
                  EMAIL
                </label>
                <Input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="node@aetherius.io"
                  required
                  className="h-10"
                  style={{
                    border: '1px solid rgba(0, 119, 182, 0.2)',
                    backgroundColor: 'rgba(0, 8, 20, 0.6)',
                    color: 'var(--surface-mist)',
                  }}
                />
              </div>

              <div>
                <label
                  className="mb-1.5 block font-mono text-xs tracking-[0.15em]"
                  style={{ color: 'rgba(202, 240, 248, 0.6)' }}
                >
                  PASSWORD
                </label>
                <Input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder={mode === 'register' ? 'Min. 8 characters' : '••••••••'}
                  required
                  minLength={8}
                  className="h-10"
                  style={{
                    border: '1px solid rgba(0, 119, 182, 0.2)',
                    backgroundColor: 'rgba(0, 8, 20, 0.6)',
                    color: 'var(--surface-mist)',
                  }}
                />
              </div>

              {error && (
                <div
                  className="rounded-md px-3 py-2 text-center font-mono text-xs tracking-wider"
                  style={{
                    color: error.includes('created') ? 'var(--glacial-cyan)' : '#ff6b6b',
                    backgroundColor: error.includes('created')
                      ? 'rgba(144, 224, 239, 0.1)'
                      : 'rgba(255, 107, 107, 0.1)',
                    border: error.includes('created')
                      ? '1px solid rgba(144, 224, 239, 0.2)'
                      : '1px solid rgba(255, 107, 107, 0.2)',
                  }}
                >
                  {error}
                </div>
              )}

              <Button
                type="submit"
                disabled={loading}
                className="h-10 w-full font-mono text-xs tracking-[0.15em] transition-all duration-300 hover:brightness-110"
                style={{
                  backgroundColor: 'var(--core-blue)',
                  color: 'var(--surface-mist)',
                  border: '1px solid rgba(144, 224, 239, 0.2)',
                }}
              >
                {loading ? (
                  <span className="flex items-center gap-2">
                    <span className="h-1.5 w-1.5 animate-pulse rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
                    {mode === 'login' ? 'AUTHENTICATING...' : 'REGISTERING...'}
                  </span>
                ) : (
                  mode === 'login' ? 'SIGN IN' : 'CREATE ACCOUNT'
                )}
              </Button>
            </form>

            <div className="mt-6 text-center">
              <button
                onClick={() => navigate('/')}
                className="font-mono text-xs tracking-[0.15em] transition-colors duration-300 hover:brightness-125"
                style={{ color: 'rgba(144, 224, 239, 0.5)' }}
              >
                ← BACK TO NETWORK
              </button>
            </div>
          </div>

          <div className="mt-6 text-center">
            <div
              className="mx-auto h-px w-24"
              style={{
                background: 'linear-gradient(to right, transparent, rgba(0, 119, 182, 0.3), transparent)',
              }}
            />
            <p
              className="mt-4 font-mono text-[10px] tracking-[0.2em]"
              style={{ color: 'rgba(202, 240, 248, 0.3)' }}
            >
              SECURED WITH JWT + TOTP MFA
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}
