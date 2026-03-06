/**
 * Smart contract ABIs and addresses for Base mainnet
 */

// Base Mainnet addresses
export const ESCROW_ADDRESS = (import.meta.env.VITE_ESCROW_ADDRESS || '0x0000000000000000000000000000000000000000') as `0x${string}`
export const USDC_ADDRESS = (import.meta.env.VITE_USDC_ADDRESS || '0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913') as `0x${string}` // Base USDC

// USDC ABI (ERC20 minimal interface for approve/allowance/balance)
export const USDC_ABI = [
  {
    name: 'approve',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [
      { name: 'spender', type: 'address' },
      { name: 'amount', type: 'uint256' },
    ],
    outputs: [{ name: '', type: 'bool' }],
  },
  {
    name: 'allowance',
    type: 'function',
    stateMutability: 'view',
    inputs: [
      { name: 'owner', type: 'address' },
      { name: 'spender', type: 'address' },
    ],
    outputs: [{ name: '', type: 'uint256' }],
  },
  {
    name: 'balanceOf',
    type: 'function',
    stateMutability: 'view',
    inputs: [{ name: 'account', type: 'address' }],
    outputs: [{ name: '', type: 'uint256' }],
  },
] as const

// AgentEscrow ABI
export const ESCROW_ABI = [
  // --- Core functions (relayer-only, but frontend reads status) ---
  {
    name: 'deposit',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [
      { name: 'escrowId', type: 'bytes32' },
      { name: 'poster', type: 'address' },
      { name: 'agent', type: 'address' },
      { name: 'amount', type: 'uint256' },
      { name: 'bidHash', type: 'bytes32' },
    ],
    outputs: [],
  },
  {
    name: 'release',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [{ name: 'escrowId', type: 'bytes32' }],
    outputs: [],
  },
  {
    name: 'refund',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [{ name: 'escrowId', type: 'bytes32' }],
    outputs: [],
  },
  {
    name: 'split',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [
      { name: 'escrowId', type: 'bytes32' },
      { name: 'agentAmount', type: 'uint256' },
    ],
    outputs: [],
  },
  {
    name: 'reassignAgent',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [
      { name: 'escrowId', type: 'bytes32' },
      { name: 'newAgent', type: 'address' },
    ],
    outputs: [],
  },
  {
    name: 'markSubmitted',
    type: 'function',
    stateMutability: 'nonpayable',
    inputs: [{ name: 'escrowId', type: 'bytes32' }],
    outputs: [],
  },

  // --- View functions ---
  {
    name: 'getEscrow',
    type: 'function',
    stateMutability: 'view',
    inputs: [{ name: 'escrowId', type: 'bytes32' }],
    outputs: [
      {
        name: '',
        type: 'tuple',
        components: [
          { name: 'poster', type: 'address' },
          { name: 'agent', type: 'address' },
          { name: 'amount', type: 'uint256' },
          { name: 'bidHash', type: 'bytes32' },
          { name: 'createdAt', type: 'uint64' },
          { name: 'submitted', type: 'bool' },
          { name: 'status', type: 'uint8' },
        ],
      },
    ],
  },
  {
    name: 'usdc',
    type: 'function',
    stateMutability: 'view',
    inputs: [],
    outputs: [{ name: '', type: 'address' }],
  },
  {
    name: 'admin',
    type: 'function',
    stateMutability: 'view',
    inputs: [],
    outputs: [{ name: '', type: 'address' }],
  },
  {
    name: 'relayer',
    type: 'function',
    stateMutability: 'view',
    inputs: [],
    outputs: [{ name: '', type: 'address' }],
  },
  {
    name: 'treasury',
    type: 'function',
    stateMutability: 'view',
    inputs: [],
    outputs: [{ name: '', type: 'address' }],
  },
  {
    name: 'paused',
    type: 'function',
    stateMutability: 'view',
    inputs: [],
    outputs: [{ name: '', type: 'bool' }],
  },

  // --- Events (for indexing) ---
  {
    name: 'Deposited',
    type: 'event',
    inputs: [
      { name: 'escrowId', type: 'bytes32', indexed: true },
      { name: 'poster', type: 'address', indexed: true },
      { name: 'agent', type: 'address', indexed: true },
      { name: 'amount', type: 'uint256', indexed: false },
      { name: 'bidHash', type: 'bytes32', indexed: false },
    ],
  },
  {
    name: 'Released',
    type: 'event',
    inputs: [
      { name: 'escrowId', type: 'bytes32', indexed: true },
      { name: 'agent', type: 'address', indexed: true },
      { name: 'amount', type: 'uint256', indexed: false },
    ],
  },
  {
    name: 'Refunded',
    type: 'event',
    inputs: [
      { name: 'escrowId', type: 'bytes32', indexed: true },
      { name: 'poster', type: 'address', indexed: true },
      { name: 'amount', type: 'uint256', indexed: false },
    ],
  },
  {
    name: 'Split',
    type: 'event',
    inputs: [
      { name: 'escrowId', type: 'bytes32', indexed: true },
      { name: 'poster', type: 'address', indexed: true },
      { name: 'agent', type: 'address', indexed: true },
      { name: 'posterAmount', type: 'uint256', indexed: false },
      { name: 'agentAmount', type: 'uint256', indexed: false },
    ],
  },
  {
    name: 'WorkSubmitted',
    type: 'event',
    inputs: [
      { name: 'escrowId', type: 'bytes32', indexed: true },
    ],
  },
] as const
