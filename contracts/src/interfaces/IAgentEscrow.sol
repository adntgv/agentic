// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/**
 * @title IAgentEscrow
 * @notice Interface for the AgentEscrow contract (v0.1.5 spec)
 * @dev Non-upgradeable escrow for AI agent marketplace on Base
 */
interface IAgentEscrow {
    // --- Enums ---
    
    enum EscrowStatus {
        None,       // 0: Escrow does not exist
        Locked,     // 1: Funds locked, work in progress
        Released,   // 2: Funds released to agent
        Refunded,   // 3: Funds refunded to poster
        Split       // 4: Funds split between poster and agent
    }

    // --- Structs ---
    
    struct Escrow {
        address poster;          // Task poster (client)
        address agent;           // Assigned AI agent
        uint256 amount;          // Total USDC locked (6 decimals)
        bytes32 bidHash;         // Hash of accepted bid (for verification)
        uint64 createdAt;        // Timestamp when escrow created
        bool submitted;          // True if agent marked work as submitted
        EscrowStatus status;     // Current escrow state
    }

    // --- Events ---
    
    event Deposited(
        bytes32 indexed escrowId,
        address indexed poster,
        address indexed agent,
        uint256 amount,
        bytes32 bidHash
    );
    
    event Released(bytes32 indexed escrowId, address indexed agent, uint256 amount);
    event Refunded(bytes32 indexed escrowId, address indexed poster, uint256 amount);
    
    event Split(
        bytes32 indexed escrowId,
        address indexed poster,
        address indexed agent,
        uint256 posterAmount,
        uint256 agentAmount
    );
    
    event AgentReassigned(
        bytes32 indexed escrowId,
        address indexed oldAgent,
        address indexed newAgent
    );
    
    event WorkSubmitted(bytes32 indexed escrowId);
    
    event RelayerProposed(address indexed proposedRelayer, uint64 effectiveAt);
    event RelayerConfirmed(address indexed newRelayer);
    event RelayerFlagged(address indexed relayer);
    event RelayerUnflagged();
    
    event TreasuryProposed(address indexed proposedTreasury, uint64 effectiveAt);
    event TreasuryConfirmed(address indexed newTreasury);
    
    event EmergencyRefunded(bytes32 indexed escrowId, address indexed poster, uint256 amount);
    event Paused(address indexed admin);
    event Unpaused(address indexed admin);

    // --- Errors ---
    
    error Unauthorized();
    error InvalidAddress();
    error InvalidAmount();
    error EscrowAlreadyExists();
    error EscrowNotFound();
    error InvalidEscrowStatus();
    error WorkAlreadySubmitted();
    error TimelockNotExpired();
    error EmergencyRefundTooEarly();
    error ContractPaused();
    error RelayerCompromised();

    // --- Core Functions ---
    
    /**
     * @notice Deposit USDC into escrow after bid acceptance
     * @param escrowId Unique identifier for this escrow
     * @param poster Address of the task poster
     * @param agent Address of the assigned agent
     * @param amount USDC amount to lock (6 decimals)
     * @param bidHash Hash of the accepted bid (keccak256(abi.encode(bidData)))
     */
    function deposit(
        bytes32 escrowId,
        address poster,
        address agent,
        uint256 amount,
        bytes32 bidHash
    ) external;

    /**
     * @notice Release full amount to agent
     * @param escrowId Escrow to release
     */
    function release(bytes32 escrowId) external;

    /**
     * @notice Refund full amount to poster
     * @param escrowId Escrow to refund
     */
    function refund(bytes32 escrowId) external;

    /**
     * @notice Split amount between poster and agent
     * @param escrowId Escrow to split
     * @param agentAmount Amount to send to agent (remainder goes to poster)
     */
    function split(bytes32 escrowId, uint256 agentAmount) external;

    /**
     * @notice Reassign agent before work is submitted
     * @param escrowId Escrow to modify
     * @param newAgent New agent address
     */
    function reassignAgent(bytes32 escrowId, address newAgent) external;

    /**
     * @notice Mark work as submitted by agent
     * @param escrowId Escrow to mark
     */
    function markSubmitted(bytes32 escrowId) external;

    // --- Admin Functions ---
    
    /**
     * @notice Pause all escrow operations
     */
    function pause() external;

    /**
     * @notice Unpause escrow operations
     */
    function unpause() external;

    /**
     * @notice Propose new relayer (24h timelock)
     * @param newRelayer Address of proposed relayer
     */
    function proposeRelayer(address newRelayer) external;

    /**
     * @notice Confirm relayer change after timelock
     */
    function confirmRelayer() external;

    /**
     * @notice Propose new treasury address (24h timelock)
     * @param newTreasury Address of proposed treasury
     */
    function proposeTreasury(address newTreasury) external;

    /**
     * @notice Confirm treasury change after timelock
     */
    function confirmTreasury() external;

    /**
     * @notice Flag relayer as compromised (disables all relayer functions)
     */
    function flagRelayerCompromised() external;

    /**
     * @notice Unflag relayer after security incident resolved
     */
    function unflagRelayer() external;

    /**
     * @notice Emergency refund after 180 days (admin only)
     * @param escrowId Escrow to emergency refund
     */
    function emergencyRefund(bytes32 escrowId) external;

    // --- View Functions ---
    
    function getEscrow(bytes32 escrowId) external view returns (Escrow memory);
    function usdc() external view returns (address);
    function admin() external view returns (address);
    function relayer() external view returns (address);
    function treasury() external view returns (address);
    function paused() external view returns (bool);
    function relayerCompromised() external view returns (bool);
}
