/**
 * Wallet connect button with RainbowKit integration and auth flow
 */

import { ConnectButton as RainbowConnectButton } from '@rainbow-me/rainbowkit'
import { useAuth } from '@/contexts/AuthContext'
import { useEffect } from 'react'
import { useAccount } from 'wagmi'

export function ConnectButton() {
  const { isAuthenticated, login } = useAuth()
  const { isConnected } = useAccount()

  // Auto-login when wallet connects if not authenticated
  useEffect(() => {
    if (isConnected && !isAuthenticated) {
      login().catch((error: unknown) => {
        console.error('Auto-login failed:', error)
      })
    }
  }, [isConnected, isAuthenticated, login])

  return (
    <RainbowConnectButton.Custom>
      {({
        account,
        chain,
        openAccountModal,
        openChainModal,
        openConnectModal,
        mounted,
      }) => {
        const ready = mounted
        const connected = ready && account && chain

        return (
          <div
            {...(!ready && {
              'aria-hidden': true,
              style: {
                opacity: 0,
                pointerEvents: 'none',
                userSelect: 'none',
              },
            })}
          >
            {(() => {
              if (!connected) {
                return (
                  <button
                    onClick={openConnectModal}
                    type="button"
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors"
                  >
                    Connect Wallet
                  </button>
                )
              }

              if (chain.unsupported) {
                return (
                  <button
                    onClick={openChainModal}
                    type="button"
                    className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white font-medium rounded-lg transition-colors"
                  >
                    Wrong Network
                  </button>
                )
              }

              return (
                <button
                  onClick={openAccountModal}
                  type="button"
                  className="px-4 py-2 bg-gray-100 hover:bg-gray-200 text-gray-900 font-medium rounded-lg transition-colors flex items-center gap-2"
                >
                  <span className="w-2 h-2 bg-green-500 rounded-full"></span>
                  {account.displayName}
                </button>
              )
            })()}
          </div>
        )
      }}
    </RainbowConnectButton.Custom>
  )
}
