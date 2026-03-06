/**
 * Authentication context with SIWE wallet login
 */

import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { useAccount, useSignMessage } from 'wagmi'
import { SiweMessage } from 'siwe'
import { walletAuth } from '@/lib/api'
import type { WorkerProfile } from '@/types'

interface AuthContextValue {
  worker: WorkerProfile | null
  workerId: string | null
  isAdmin: boolean
  isAuthenticated: boolean
  isLoading: boolean
  login: () => Promise<void>
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export function AuthProvider({ children }: { children: ReactNode }) {
  const { address, isConnected } = useAccount()
  const { signMessageAsync } = useSignMessage()
  const [worker, setWorker] = useState<WorkerProfile | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  const token = localStorage.getItem('auth_token')
  const isAuthenticated = !!token && !!worker

  // Check for existing auth on mount
  useEffect(() => {
    if (token && address) {
      // TODO: fetch worker profile from /workers/me endpoint
      // For now, just set a placeholder
      setWorker({
        id: 'placeholder',
        worker_type: 'user',
        wallet_address: address,
        status: 'active',
        created_at: new Date().toISOString(),
      })
    }
  }, [token, address])

  const login = async () => {
    if (!address) throw new Error('Wallet not connected')
    setIsLoading(true)

    try {
      const siweMessage = new SiweMessage({
        domain: window.location.host,
        address,
        statement: 'Sign in to Agentic Marketplace',
        uri: window.location.origin,
        version: '1',
        chainId: 8453, // Base mainnet
        nonce: crypto.randomUUID(),
      })

      const message = siweMessage.prepareMessage()
      const signature = await signMessageAsync({ message })
      const { token } = await walletAuth(address, signature, message)

      localStorage.setItem('auth_token', token)

      // TODO: fetch worker profile
      setWorker({
        id: 'placeholder',
        worker_type: 'user',
        wallet_address: address,
        status: 'active',
        created_at: new Date().toISOString(),
      })
    } catch (error) {
      console.error('Login failed:', error)
      throw error
    } finally {
      setIsLoading(false)
    }
  }

  const logout = () => {
    localStorage.removeItem('auth_token')
    setWorker(null)
  }

  return (
    <AuthContext.Provider value={{ worker, workerId: worker?.id ?? null, isAdmin: (worker as any)?.is_admin === true, isAuthenticated, isLoading, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
