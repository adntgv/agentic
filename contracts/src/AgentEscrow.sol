// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "./interfaces/IAgentEscrow.sol";

/**
 * @title AgentEscrow
 * @notice Non-upgradeable escrow contract for AI agent marketplace on Base
 * @dev Implements v0.1.5 specification with relayer-based settlement
 * @custom:security-contact security@agentic.example
 */
contract AgentEscrow is IAgentEscrow, ReentrancyGuard {
    using SafeERC20 for IERC20;

    // --- Constants ---
    
    uint64 public constant TIMELOCK_DURATION = 24 hours;
    uint64 public constant EMERGENCY_REFUND_DELAY = 180 days;

    // --- Storage ---
    
    IERC20 private immutable _usdc;
    address public admin;
    address public relayer;
    address public treasury;
    
    bool public paused;
    bool public relayerCompromised;
    
    mapping(bytes32 => Escrow) private escrows;
    
    // Timelock for relayer rotation
    address public proposedRelayer;
    uint64 public relayerEffectiveAt;
    
    // Timelock for treasury rotation
    address public proposedTreasury;
    uint64 public treasuryEffectiveAt;

    // --- Modifiers ---
    
    modifier onlyAdmin() {
        if (msg.sender != admin) revert Unauthorized();
        _;
    }

    modifier onlyRelayer() {
        if (msg.sender != relayer) revert Unauthorized();
        _;
    }

    modifier whenNotPaused() {
        if (paused) revert ContractPaused();
        _;
    }

    modifier whenRelayerNotCompromised() {
        if (relayerCompromised) revert RelayerCompromised();
        _;
    }

    // --- Constructor ---
    
    /**
     * @notice Deploy the escrow contract
     * @param usdcAddress USDC token address on Base
     * @param relayerAddress Initial relayer address
     * @param treasuryAddress Initial treasury address
     */
    constructor(address usdcAddress, address relayerAddress, address treasuryAddress) {
        if (usdcAddress == address(0)) revert InvalidAddress();
        if (relayerAddress == address(0)) revert InvalidAddress();
        if (treasuryAddress == address(0)) revert InvalidAddress();

        _usdc = IERC20(usdcAddress);
        admin = msg.sender;
        relayer = relayerAddress;
        treasury = treasuryAddress;
    }

    // --- Core Functions ---
    
    /**
     * @inheritdoc IAgentEscrow
     */
    function deposit(
        bytes32 escrowId,
        address poster,
        address agent,
        uint256 amount,
        bytes32 bidHash
    ) external onlyRelayer whenNotPaused whenRelayerNotCompromised nonReentrant {
        if (poster == address(0)) revert InvalidAddress();
        if (agent == address(0)) revert InvalidAddress();
        if (amount == 0) revert InvalidAmount();
        if (escrows[escrowId].status != EscrowStatus.None) revert EscrowAlreadyExists();

        escrows[escrowId] = Escrow({
            poster: poster,
            agent: agent,
            amount: amount,
            bidHash: bidHash,
            createdAt: uint64(block.timestamp),
            submitted: false,
            status: EscrowStatus.Locked
        });

        _usdc.safeTransferFrom(poster, address(this), amount);

        emit Deposited(escrowId, poster, agent, amount, bidHash);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function release(bytes32 escrowId) 
        external 
        onlyRelayer 
        whenNotPaused 
        whenRelayerNotCompromised 
        nonReentrant 
    {
        Escrow storage e = escrows[escrowId];
        if (e.status != EscrowStatus.Locked) {
            if (e.status == EscrowStatus.None) revert EscrowNotFound();
            revert InvalidEscrowStatus();
        }

        e.status = EscrowStatus.Released;
        _usdc.safeTransfer(e.agent, e.amount);

        emit Released(escrowId, e.agent, e.amount);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function refund(bytes32 escrowId) 
        external 
        onlyRelayer 
        whenNotPaused 
        whenRelayerNotCompromised 
        nonReentrant 
    {
        Escrow storage e = escrows[escrowId];
        if (e.status != EscrowStatus.Locked) {
            if (e.status == EscrowStatus.None) revert EscrowNotFound();
            revert InvalidEscrowStatus();
        }

        e.status = EscrowStatus.Refunded;
        _usdc.safeTransfer(e.poster, e.amount);

        emit Refunded(escrowId, e.poster, e.amount);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function split(bytes32 escrowId, uint256 agentAmount) 
        external 
        onlyRelayer 
        whenNotPaused 
        whenRelayerNotCompromised 
        nonReentrant 
    {
        Escrow storage e = escrows[escrowId];
        if (e.status != EscrowStatus.Locked) {
            if (e.status == EscrowStatus.None) revert EscrowNotFound();
            revert InvalidEscrowStatus();
        }
        if (agentAmount > e.amount) revert InvalidAmount();

        uint256 posterAmount = e.amount - agentAmount;

        e.status = EscrowStatus.Split;

        if (posterAmount > 0) {
            _usdc.safeTransfer(e.poster, posterAmount);
        }
        if (agentAmount > 0) {
            _usdc.safeTransfer(e.agent, agentAmount);
        }

        emit Split(escrowId, e.poster, e.agent, posterAmount, agentAmount);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function reassignAgent(bytes32 escrowId, address newAgent) 
        external 
        onlyRelayer 
        whenNotPaused 
        whenRelayerNotCompromised 
        nonReentrant 
    {
        if (newAgent == address(0)) revert InvalidAddress();
        
        Escrow storage e = escrows[escrowId];
        if (e.status != EscrowStatus.Locked) {
            if (e.status == EscrowStatus.None) revert EscrowNotFound();
            revert InvalidEscrowStatus();
        }
        if (e.submitted) revert WorkAlreadySubmitted();

        address oldAgent = e.agent;
        e.agent = newAgent;

        emit AgentReassigned(escrowId, oldAgent, newAgent);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function markSubmitted(bytes32 escrowId) 
        external 
        onlyRelayer 
        whenNotPaused 
        whenRelayerNotCompromised 
        nonReentrant 
    {
        Escrow storage e = escrows[escrowId];
        if (e.status != EscrowStatus.Locked) {
            if (e.status == EscrowStatus.None) revert EscrowNotFound();
            revert InvalidEscrowStatus();
        }
        if (e.submitted) revert WorkAlreadySubmitted();

        e.submitted = true;

        emit WorkSubmitted(escrowId);
    }

    // --- Admin Functions ---
    
    /**
     * @inheritdoc IAgentEscrow
     */
    function pause() external onlyAdmin {
        paused = true;
        emit Paused(msg.sender);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function unpause() external onlyAdmin {
        paused = false;
        emit Unpaused(msg.sender);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function proposeRelayer(address newRelayer) external onlyAdmin {
        if (newRelayer == address(0)) revert InvalidAddress();
        
        proposedRelayer = newRelayer;
        relayerEffectiveAt = uint64(block.timestamp + TIMELOCK_DURATION);

        emit RelayerProposed(newRelayer, relayerEffectiveAt);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function confirmRelayer() external onlyAdmin {
        if (block.timestamp < relayerEffectiveAt) revert TimelockNotExpired();
        
        relayer = proposedRelayer;
        proposedRelayer = address(0);
        relayerEffectiveAt = 0;

        emit RelayerConfirmed(relayer);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function proposeTreasury(address newTreasury) external onlyAdmin {
        if (newTreasury == address(0)) revert InvalidAddress();
        
        proposedTreasury = newTreasury;
        treasuryEffectiveAt = uint64(block.timestamp + TIMELOCK_DURATION);

        emit TreasuryProposed(newTreasury, treasuryEffectiveAt);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function confirmTreasury() external onlyAdmin {
        if (block.timestamp < treasuryEffectiveAt) revert TimelockNotExpired();
        
        treasury = proposedTreasury;
        proposedTreasury = address(0);
        treasuryEffectiveAt = 0;

        emit TreasuryConfirmed(treasury);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function flagRelayerCompromised() external onlyAdmin {
        relayerCompromised = true;
        emit RelayerFlagged(relayer);
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function unflagRelayer() external onlyAdmin {
        relayerCompromised = false;
        emit RelayerUnflagged();
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function emergencyRefund(bytes32 escrowId) external onlyAdmin nonReentrant {
        Escrow storage e = escrows[escrowId];
        if (e.status != EscrowStatus.Locked) {
            if (e.status == EscrowStatus.None) revert EscrowNotFound();
            revert InvalidEscrowStatus();
        }
        if (block.timestamp < e.createdAt + EMERGENCY_REFUND_DELAY) {
            revert EmergencyRefundTooEarly();
        }

        e.status = EscrowStatus.Refunded;
        _usdc.safeTransfer(e.poster, e.amount);

        emit EmergencyRefunded(escrowId, e.poster, e.amount);
    }

    // --- View Functions ---
    
    /**
     * @inheritdoc IAgentEscrow
     */
    function getEscrow(bytes32 escrowId) external view returns (Escrow memory) {
        return escrows[escrowId];
    }

    /**
     * @inheritdoc IAgentEscrow
     */
    function usdc() external view returns (address) {
        return address(_usdc);
    }
}
