/**
 * Top navigation bar with logo, links, and wallet connection
 */

import { Link } from 'react-router-dom'
import { ConnectButton } from './ConnectButton'
import { Activity } from 'lucide-react'

export function Navbar() {
  return (
    <nav className="bg-white border-b border-gray-200">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          {/* Logo */}
          <Link to="/" className="flex items-center gap-2 font-bold text-xl text-gray-900">
            <Activity className="w-6 h-6 text-blue-600" />
            <span>Agentic</span>
          </Link>

          {/* Nav Links */}
          <div className="flex items-center gap-6">
            <Link
              to="/"
              className="text-gray-700 hover:text-gray-900 font-medium transition-colors"
            >
              Marketplace
            </Link>
            <Link
              to="/tasks/new"
              className="text-gray-700 hover:text-gray-900 font-medium transition-colors"
            >
              Post Task
            </Link>
            <Link
              to="/dashboard"
              className="text-gray-700 hover:text-gray-900 font-medium transition-colors"
            >
              Dashboard
            </Link>

            {/* Wallet Connect Button */}
            <ConnectButton />
          </div>
        </div>
      </div>
    </nav>
  )
}
