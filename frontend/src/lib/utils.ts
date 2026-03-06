/**
 * Utility functions for the frontend
 */

import { type ClassValue, clsx } from 'clsx'

/**
 * Merge tailwind classes with clsx
 */
export function cn(...inputs: ClassValue[]) {
  return clsx(inputs)
}

/**
 * Format USDC amount (6 decimals) to display string
 */
export function formatUSDC(amount: string | number): string {
  const num = typeof amount === 'string' ? parseFloat(amount) : amount
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(num)
}

/**
 * Truncate ethereum address for display
 */
export function truncateAddress(address: string, chars = 4): string {
  if (!address) return ''
  return `${address.slice(0, chars + 2)}...${address.slice(-chars)}`
}

/**
 * Format relative time (e.g., "2 hours ago")
 */
export function formatRelativeTime(date: string | Date): string {
  const now = new Date()
  const then = new Date(date)
  const seconds = Math.floor((now.getTime() - then.getTime()) / 1000)

  if (seconds < 60) return 'just now'
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`
  return then.toLocaleDateString()
}

/**
 * Parse USDC string to wei (6 decimals)
 */
export function parseUSDC(amount: string): bigint {
  const num = parseFloat(amount)
  return BigInt(Math.floor(num * 1_000_000))
}

/**
 * Format wei to USDC string (6 decimals)
 */
export function formatUSDCFromWei(wei: bigint): string {
  return (Number(wei) / 1_000_000).toFixed(2)
}
