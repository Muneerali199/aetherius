import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import FluidBackground from '@/components/effects/FluidBackground'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

interface Wallet {
  id: string
  balance: number
  currency: string
}

interface Transaction {
  id: string
  type: string
  status: string
  amount: number
  net_amount: number
  balance_after: number
  description: string
  created_at: string
}

interface PaymentMethod {
  id: string
  brand: string
  last4: string
  exp_month: number
  exp_year: number
  is_default: boolean
}

export default function Billing() {
  const [wallet, setWallet] = useState<Wallet | null>(null)
  const [txns, setTxns] = useState<Transaction[]>([])
  const [pms, setPms] = useState<PaymentMethod[]>([])
  const [loading, setLoading] = useState(true)
  const [topUpAmount, setTopUpAmount] = useState('')
  const navigate = useNavigate()

  const headers = () => ({ Authorization: `Bearer ${localStorage.getItem('access_token')}` })

  useEffect(() => {
    const token = localStorage.getItem('access_token')
    if (!token) { navigate('/login'); return }

    Promise.all([
      fetch('/v1/payments/wallet', { headers: headers() }).then(r => r.ok ? r.json() : null),
      fetch('/v1/payments/transactions', { headers: headers() }).then(r => r.ok ? r.json() : { transactions: [] }),
      fetch('/v1/payments/methods', { headers: headers() }).then(r => r.ok ? r.json() : { payment_methods: [] }),
    ])
      .then(([w, t, p]) => {
        if (w) setWallet(w)
        setTxns(t.transactions || [])
        setPms(p.payment_methods || [])
      })
      .catch(() => navigate('/login'))
      .finally(() => setLoading(false))
  }, [navigate])

  const handleTopUp = async () => {
    const amount = parseInt(topUpAmount)
    if (!amount || amount < 1) return

    const res = await fetch('/v1/payments/create-intent', {
      method: 'POST',
      headers: { ...headers(), 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount: amount * 100, currency: 'usd', description: 'Wallet top-up' }),
    })
    const data = await res.json()

    if (data.client_secret) {
      alert(`Payment intent created!\n\nIn production, this would open Stripe Checkout.\n\nFor demo: Payment confirmed!`)
      const w = await fetch('/v1/payments/wallet', { headers: headers() }).then(r => r.json())
      setWallet(w)
      const t = await fetch('/v1/payments/transactions', { headers: headers() }).then(r => r.json())
      setTxns(t.transactions || [])
      setTopUpAmount('')
    }
  }

  if (loading) {
    return (
      <div className="relative flex min-h-screen items-center justify-center" style={{ backgroundColor: 'var(--abyssal-black)' }}>
        <FluidBackground />
        <div className="relative flex items-center gap-3">
          <span className="h-2 w-2 animate-pulse rounded-full" style={{ backgroundColor: 'var(--glacial-cyan)' }} />
          <span className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.5)' }}>LOADING BILLING...</span>
        </div>
      </div>
    )
  }

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
            <a href="/dashboard" className="font-mono text-xs tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.6)' }}>
              ← DASHBOARD
            </a>
          </div>
        </div>
      </nav>

      <div className="relative px-6 pt-32 pb-20 md:px-12">
        <div className="mx-auto max-w-[1200px]">
          <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>BILLING</p>
          <h1 className="mt-2 font-sans text-[clamp(1.5rem,3vw,2.5rem)] font-semibold leading-[1.1] tracking-[-0.02em]" style={{ color: 'var(--glacial-cyan)' }}>
            Wallet & Payments
          </h1>

          <div className="mt-8 grid gap-6 md:grid-cols-3">
            <div className="rounded-xl p-6 md:col-span-1" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
              <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>BALANCE</p>
              <p className="mt-3 font-sans text-4xl font-semibold tracking-tight" style={{ color: 'var(--glacial-cyan)' }}>
                ${(wallet?.balance || 0).toFixed(2)}
              </p>
              <p className="mt-1 font-mono text-[10px] tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>
                {wallet?.currency || 'USD'} · Available for deployments
              </p>

              <div className="mt-6 space-y-3">
                <Input
                  type="number"
                  min="1"
                  step="1"
                  placeholder="Amount ($)"
                  value={topUpAmount}
                  onChange={(e) => setTopUpAmount(e.target.value)}
                  className="h-10"
                  style={{ border: '1px solid rgba(0, 119, 182, 0.2)', backgroundColor: 'rgba(0, 8, 20, 0.6)', color: 'var(--surface-mist)' }}
                />
                <Button
                  onClick={handleTopUp}
                  disabled={!topUpAmount || parseInt(topUpAmount) < 1}
                  className="h-10 w-full font-mono text-xs tracking-[0.15em]"
                  style={{ backgroundColor: 'var(--core-blue)', color: 'var(--surface-mist)' }}
                >
                  ADD FUNDS
                </Button>
              </div>

              <div className="mt-6 rounded-lg p-3" style={{ backgroundColor: 'rgba(0, 119, 182, 0.06)' }}>
                <p className="font-mono text-[10px] tracking-wider" style={{ color: 'rgba(144, 224, 239, 0.4)' }}>
                  PAYMENT METHODS ({pms.length})
                </p>
                {pms.length === 0 ? (
                  <p className="mt-2 font-mono text-xs" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>No cards saved</p>
                ) : (
                  <div className="mt-2 space-y-2">
                    {pms.map(pm => (
                      <div key={pm.id} className="flex items-center gap-2 rounded-md px-3 py-2" style={{ backgroundColor: 'rgba(0, 119, 182, 0.08)' }}>
                        <span className="font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>
                          {pm.brand} ···· {pm.last4}
                        </span>
                        {pm.is_default && (
                          <span className="rounded px-1.5 py-0.5 font-mono text-[9px]" style={{ backgroundColor: 'rgba(144, 224, 239, 0.1)', color: 'var(--glacial-cyan)' }}>
                            DEFAULT
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>

            <div className="rounded-xl p-6 md:col-span-2" style={{ border: '1px solid rgba(0, 119, 182, 0.15)', backdropFilter: 'blur(12px)', background: 'rgba(0, 13, 29, 0.6)' }}>
              <p className="font-mono text-xs tracking-[0.2em]" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>TRANSACTION HISTORY</p>
              {txns.length === 0 ? (
                <div className="mt-12 text-center">
                  <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full" style={{ border: '1px solid rgba(144, 224, 239, 0.15)' }}>
                    <div className="h-2 w-2 rounded-full" style={{ backgroundColor: 'rgba(144, 224, 239, 0.3)' }} />
                  </div>
                  <p className="mt-3 font-mono text-xs tracking-wider" style={{ color: 'rgba(202, 240, 248, 0.4)' }}>
                    No transactions yet
                  </p>
                </div>
              ) : (
                <div className="mt-4 space-y-1">
                  {txns.map(txn => (
                    <div key={txn.id} className="flex items-center justify-between rounded-lg px-4 py-3" style={{ border: '1px solid rgba(0, 119, 182, 0.08)' }}>
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-xs" style={{ color: 'var(--surface-mist)' }}>
                            {txn.type.toUpperCase()}
                          </span>
                          <span className={`rounded px-1.5 py-0.5 font-mono text-[9px] ${
                            txn.status === 'completed' ? 'text-green-400' :
                            txn.status === 'pending' ? 'text-yellow-400' : 'text-red-400'
                          }`} style={{ backgroundColor: txn.status === 'completed' ? 'rgba(34,197,94,0.1)' : 'rgba(234,179,8,0.1)' }}>
                            {txn.status}
                          </span>
                        </div>
                        <p className="mt-0.5 font-mono text-[10px]" style={{ color: 'rgba(202, 240, 248, 0.3)' }}>
                          {txn.description || txn.reference}
                        </p>
                      </div>
                      <div className="text-right">
                        <p className={`font-mono text-sm ${txn.net_amount >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                          {txn.net_amount >= 0 ? '+' : ''}${txn.amount.toFixed(2)}
                        </p>
                        <p className="font-mono text-[9px]" style={{ color: 'rgba(202, 240, 248, 0.2)' }}>
                          Balance: ${txn.balance_after.toFixed(2)}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
