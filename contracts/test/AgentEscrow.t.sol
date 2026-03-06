// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/AgentEscrow.sol";
import "../src/interfaces/IAgentEscrow.sol";
import "./mocks/MockUSDC.sol";

contract AgentEscrowTest is Test {
    AgentEscrow public escrow;
    MockUSDC public usdc;

    address public admin = makeAddr("admin");
    address public relayer = makeAddr("relayer");
    address public treasury = makeAddr("treasury");
    address public poster = makeAddr("poster");
    address public agent = makeAddr("agent");
    address public newAgent = makeAddr("newAgent");

    bytes32 public constant ESCROW_ID = keccak256("task1");
    bytes32 public constant BID_HASH = keccak256("bid_data_1");
    uint256 public constant AMOUNT = 100 * 1e6; // 100 USDC

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
    event Paused(address indexed admin);
    event Unpaused(address indexed admin);
    event RelayerProposed(address indexed proposedRelayer, uint64 effectiveAt);
    event RelayerConfirmed(address indexed newRelayer);
    event RelayerFlagged(address indexed relayer);
    event RelayerUnflagged();
    event TreasuryProposed(address indexed proposedTreasury, uint64 effectiveAt);
    event TreasuryConfirmed(address indexed newTreasury);
    event EmergencyRefunded(bytes32 indexed escrowId, address indexed poster, uint256 amount);

    function setUp() public {
        usdc = new MockUSDC();
        
        vm.startPrank(admin);
        escrow = new AgentEscrow(address(usdc), relayer, treasury);
        vm.stopPrank();

        // Mint USDC to poster
        usdc.mint(poster, 1000 * 1e6);
        
        // Approve escrow
        vm.prank(poster);
        usdc.approve(address(escrow), type(uint256).max);
    }

    // ============================================
    // 1. DEPLOYMENT AND INITIAL STATE
    // ============================================

    function test_Deployment() public view {
        assertEq(escrow.admin(), admin);
        assertEq(escrow.relayer(), relayer);
        assertEq(escrow.treasury(), treasury);
        assertEq(address(escrow.usdc()), address(usdc));
        assertEq(escrow.paused(), false);
        assertEq(escrow.relayerCompromised(), false);
    }

    function test_Deployment_RevertIf_ZeroAddress() public {
        vm.startPrank(admin);
        
        vm.expectRevert(IAgentEscrow.InvalidAddress.selector);
        new AgentEscrow(address(0), relayer, treasury);
        
        vm.expectRevert(IAgentEscrow.InvalidAddress.selector);
        new AgentEscrow(address(usdc), address(0), treasury);
        
        vm.expectRevert(IAgentEscrow.InvalidAddress.selector);
        new AgentEscrow(address(usdc), relayer, address(0));
        
        vm.stopPrank();
    }

    // ============================================
    // 2. DEPOSIT FLOW
    // ============================================

    function test_Deposit_Success() public {
        vm.prank(relayer);
        
        vm.expectEmit(true, true, true, true);
        emit Deposited(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);
        
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        IAgentEscrow.Escrow memory e = escrow.getEscrow(ESCROW_ID);
        assertEq(e.poster, poster);
        assertEq(e.agent, agent);
        assertEq(e.amount, AMOUNT);
        assertEq(e.bidHash, BID_HASH);
        assertEq(e.submitted, false);
        assertEq(uint8(e.status), uint8(IAgentEscrow.EscrowStatus.Locked));
        assertEq(usdc.balanceOf(address(escrow)), AMOUNT);
    }

    function test_Deposit_RevertIf_NotRelayer() public {
        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.Unauthorized.selector);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);
    }

    function test_Deposit_RevertIf_Paused() public {
        vm.prank(admin);
        escrow.pause();

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.ContractPaused.selector);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);
    }

    function test_Deposit_RevertIf_RelayerCompromised() public {
        vm.prank(admin);
        escrow.flagRelayerCompromised();

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.RelayerCompromised.selector);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);
    }

    function test_Deposit_RevertIf_EscrowExists() public {
        vm.startPrank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);
        
        vm.expectRevert(IAgentEscrow.EscrowAlreadyExists.selector);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);
        vm.stopPrank();
    }

    function test_Deposit_RevertIf_ZeroAmount() public {
        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.InvalidAmount.selector);
        escrow.deposit(ESCROW_ID, poster, agent, 0, BID_HASH);
    }

    function test_Deposit_RevertIf_ZeroAddress() public {
        vm.startPrank(relayer);
        
        vm.expectRevert(IAgentEscrow.InvalidAddress.selector);
        escrow.deposit(ESCROW_ID, address(0), agent, AMOUNT, BID_HASH);
        
        vm.expectRevert(IAgentEscrow.InvalidAddress.selector);
        escrow.deposit(ESCROW_ID, poster, address(0), AMOUNT, BID_HASH);
        
        vm.stopPrank();
    }

    // ============================================
    // 3. RELEASE FLOW
    // ============================================

    function test_Release_Success() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        uint256 agentBalBefore = usdc.balanceOf(agent);

        vm.prank(relayer);
        vm.expectEmit(true, true, false, true);
        emit Released(ESCROW_ID, agent, AMOUNT);
        escrow.release(ESCROW_ID);

        IAgentEscrow.Escrow memory e = escrow.getEscrow(ESCROW_ID);
        assertEq(uint8(e.status), uint8(IAgentEscrow.EscrowStatus.Released));
        assertEq(usdc.balanceOf(agent), agentBalBefore + AMOUNT);
        assertEq(usdc.balanceOf(address(escrow)), 0);
    }

    function test_Release_RevertIf_NotRelayer() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.Unauthorized.selector);
        escrow.release(ESCROW_ID);
    }

    function test_Release_RevertIf_Paused() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(admin);
        escrow.pause();

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.ContractPaused.selector);
        escrow.release(ESCROW_ID);
    }

    function test_Release_RevertIf_RelayerCompromised() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(admin);
        escrow.flagRelayerCompromised();

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.RelayerCompromised.selector);
        escrow.release(ESCROW_ID);
    }

    function test_Release_RevertIf_InvalidStatus() public {
        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.EscrowNotFound.selector);
        escrow.release(ESCROW_ID);
    }

    // ============================================
    // 4. REFUND FLOW
    // ============================================

    function test_Refund_Success() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        uint256 posterBalBefore = usdc.balanceOf(poster);

        vm.prank(relayer);
        vm.expectEmit(true, true, false, true);
        emit Refunded(ESCROW_ID, poster, AMOUNT);
        escrow.refund(ESCROW_ID);

        IAgentEscrow.Escrow memory e = escrow.getEscrow(ESCROW_ID);
        assertEq(uint8(e.status), uint8(IAgentEscrow.EscrowStatus.Refunded));
        assertEq(usdc.balanceOf(poster), posterBalBefore + AMOUNT);
        assertEq(usdc.balanceOf(address(escrow)), 0);
    }

    function test_Refund_RevertIf_NotRelayer() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.Unauthorized.selector);
        escrow.refund(ESCROW_ID);
    }

    // ============================================
    // 5. SPLIT FLOW
    // ============================================

    function test_Split_Success() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        uint256 agentAmount = 60 * 1e6;
        uint256 posterAmount = AMOUNT - agentAmount;

        uint256 posterBalBefore = usdc.balanceOf(poster);
        uint256 agentBalBefore = usdc.balanceOf(agent);

        vm.prank(relayer);
        vm.expectEmit(true, true, true, true);
        emit Split(ESCROW_ID, poster, agent, posterAmount, agentAmount);
        escrow.split(ESCROW_ID, agentAmount);

        IAgentEscrow.Escrow memory e = escrow.getEscrow(ESCROW_ID);
        assertEq(uint8(e.status), uint8(IAgentEscrow.EscrowStatus.Split));
        assertEq(usdc.balanceOf(poster), posterBalBefore + posterAmount);
        assertEq(usdc.balanceOf(agent), agentBalBefore + agentAmount);
        assertEq(usdc.balanceOf(address(escrow)), 0);
    }

    function test_Split_RevertIf_InvalidAmount() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.InvalidAmount.selector);
        escrow.split(ESCROW_ID, AMOUNT + 1);
    }

    function test_Split_ZeroToAgent() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        uint256 posterBalBefore = usdc.balanceOf(poster);

        vm.prank(relayer);
        escrow.split(ESCROW_ID, 0);

        assertEq(usdc.balanceOf(poster), posterBalBefore + AMOUNT);
        assertEq(usdc.balanceOf(agent), 0);
    }

    function test_Split_AllToAgent() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        uint256 agentBalBefore = usdc.balanceOf(agent);

        vm.prank(relayer);
        escrow.split(ESCROW_ID, AMOUNT);

        assertEq(usdc.balanceOf(agent), agentBalBefore + AMOUNT);
        assertEq(usdc.balanceOf(poster) - (1000 * 1e6 - AMOUNT), 0); // poster's balance unchanged except initial deposit
    }

    // ============================================
    // 6. REASSIGN AGENT
    // ============================================

    function test_ReassignAgent_Success() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(relayer);
        vm.expectEmit(true, true, true, false);
        emit AgentReassigned(ESCROW_ID, agent, newAgent);
        escrow.reassignAgent(ESCROW_ID, newAgent);

        IAgentEscrow.Escrow memory e = escrow.getEscrow(ESCROW_ID);
        assertEq(e.agent, newAgent);
    }

    function test_ReassignAgent_RevertIf_AlreadySubmitted() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(relayer);
        escrow.markSubmitted(ESCROW_ID);

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.WorkAlreadySubmitted.selector);
        escrow.reassignAgent(ESCROW_ID, newAgent);
    }

    function test_ReassignAgent_RevertIf_ZeroAddress() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.InvalidAddress.selector);
        escrow.reassignAgent(ESCROW_ID, address(0));
    }

    // ============================================
    // 7. MARK SUBMITTED
    // ============================================

    function test_MarkSubmitted_Success() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(relayer);
        vm.expectEmit(true, false, false, false);
        emit WorkSubmitted(ESCROW_ID);
        escrow.markSubmitted(ESCROW_ID);

        IAgentEscrow.Escrow memory e = escrow.getEscrow(ESCROW_ID);
        assertEq(e.submitted, true);
    }

    function test_MarkSubmitted_RevertIf_NotRelayer() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.Unauthorized.selector);
        escrow.markSubmitted(ESCROW_ID);
    }

    function test_MarkSubmitted_RevertIf_AlreadySubmitted() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(relayer);
        escrow.markSubmitted(ESCROW_ID);

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.WorkAlreadySubmitted.selector);
        escrow.markSubmitted(ESCROW_ID);
    }

    // ============================================
    // 8. PAUSE/UNPAUSE
    // ============================================

    function test_Pause_Success() public {
        vm.prank(admin);
        vm.expectEmit(true, false, false, false);
        emit Paused(admin);
        escrow.pause();

        assertEq(escrow.paused(), true);
    }

    function test_Unpause_Success() public {
        vm.prank(admin);
        escrow.pause();

        vm.prank(admin);
        vm.expectEmit(true, false, false, false);
        emit Unpaused(admin);
        escrow.unpause();

        assertEq(escrow.paused(), false);
    }

    function test_Pause_RevertIf_NotAdmin() public {
        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.Unauthorized.selector);
        escrow.pause();
    }

    function test_Unpause_RevertIf_NotAdmin() public {
        vm.prank(admin);
        escrow.pause();

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.Unauthorized.selector);
        escrow.unpause();
    }

    // ============================================
    // 9. RELAYER ROTATION (24H TIMELOCK)
    // ============================================

    function test_ProposeRelayer_Success() public {
        address newRelayer = makeAddr("newRelayer");
        
        vm.prank(admin);
        vm.expectEmit(true, false, false, false);
        emit RelayerProposed(newRelayer, uint64(block.timestamp + 24 hours));
        escrow.proposeRelayer(newRelayer);
    }

    function test_ConfirmRelayer_Success() public {
        address newRelayer = makeAddr("newRelayer");
        
        vm.prank(admin);
        escrow.proposeRelayer(newRelayer);

        vm.warp(block.timestamp + 24 hours);

        vm.prank(admin);
        vm.expectEmit(true, false, false, false);
        emit RelayerConfirmed(newRelayer);
        escrow.confirmRelayer();

        assertEq(escrow.relayer(), newRelayer);
    }

    function test_ConfirmRelayer_RevertIf_TimelockNotExpired() public {
        address newRelayer = makeAddr("newRelayer");
        
        vm.prank(admin);
        escrow.proposeRelayer(newRelayer);

        vm.warp(block.timestamp + 12 hours);

        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.TimelockNotExpired.selector);
        escrow.confirmRelayer();
    }

    function test_ProposeRelayer_RevertIf_NotAdmin() public {
        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.Unauthorized.selector);
        escrow.proposeRelayer(makeAddr("newRelayer"));
    }

    function test_ProposeRelayer_RevertIf_ZeroAddress() public {
        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.InvalidAddress.selector);
        escrow.proposeRelayer(address(0));
    }

    // ============================================
    // 10. TREASURY ROTATION (24H TIMELOCK)
    // ============================================

    function test_ProposeTreasury_Success() public {
        address newTreasury = makeAddr("newTreasury");
        
        vm.prank(admin);
        vm.expectEmit(true, false, false, false);
        emit TreasuryProposed(newTreasury, uint64(block.timestamp + 24 hours));
        escrow.proposeTreasury(newTreasury);
    }

    function test_ConfirmTreasury_Success() public {
        address newTreasury = makeAddr("newTreasury");
        
        vm.prank(admin);
        escrow.proposeTreasury(newTreasury);

        vm.warp(block.timestamp + 24 hours);

        vm.prank(admin);
        vm.expectEmit(true, false, false, false);
        emit TreasuryConfirmed(newTreasury);
        escrow.confirmTreasury();

        assertEq(escrow.treasury(), newTreasury);
    }

    function test_ConfirmTreasury_RevertIf_TimelockNotExpired() public {
        address newTreasury = makeAddr("newTreasury");
        
        vm.prank(admin);
        escrow.proposeTreasury(newTreasury);

        vm.warp(block.timestamp + 12 hours);

        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.TimelockNotExpired.selector);
        escrow.confirmTreasury();
    }

    function test_ProposeTreasury_RevertIf_ZeroAddress() public {
        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.InvalidAddress.selector);
        escrow.proposeTreasury(address(0));
    }

    // ============================================
    // 11. FLAG/UNFLAG RELAYER
    // ============================================

    function test_FlagRelayer_Success() public {
        vm.prank(admin);
        vm.expectEmit(true, false, false, false);
        emit RelayerFlagged(relayer);
        escrow.flagRelayerCompromised();

        assertEq(escrow.relayerCompromised(), true);
    }

    function test_UnflagRelayer_Success() public {
        vm.prank(admin);
        escrow.flagRelayerCompromised();

        vm.prank(admin);
        vm.expectEmit(false, false, false, false);
        emit RelayerUnflagged();
        escrow.unflagRelayer();

        assertEq(escrow.relayerCompromised(), false);
    }

    function test_FlagRelayer_RevertIf_NotAdmin() public {
        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.Unauthorized.selector);
        escrow.flagRelayerCompromised();
    }

    // ============================================
    // 12. EMERGENCY REFUND (180 DAYS)
    // ============================================

    function test_EmergencyRefund_Success() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.warp(block.timestamp + 180 days);

        uint256 posterBalBefore = usdc.balanceOf(poster);

        vm.prank(admin);
        vm.expectEmit(true, true, false, true);
        emit EmergencyRefunded(ESCROW_ID, poster, AMOUNT);
        escrow.emergencyRefund(ESCROW_ID);

        IAgentEscrow.Escrow memory e = escrow.getEscrow(ESCROW_ID);
        assertEq(uint8(e.status), uint8(IAgentEscrow.EscrowStatus.Refunded));
        assertEq(usdc.balanceOf(poster), posterBalBefore + AMOUNT);
    }

    function test_EmergencyRefund_RevertIf_TooEarly() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.warp(block.timestamp + 179 days);

        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.EmergencyRefundTooEarly.selector);
        escrow.emergencyRefund(ESCROW_ID);
    }

    function test_EmergencyRefund_RevertIf_NotAdmin() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.warp(block.timestamp + 180 days);

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.Unauthorized.selector);
        escrow.emergencyRefund(ESCROW_ID);
    }

    function test_EmergencyRefund_RevertIf_AlreadySettled() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(relayer);
        escrow.release(ESCROW_ID);

        vm.warp(block.timestamp + 180 days);

        vm.prank(admin);
        vm.expectRevert(IAgentEscrow.InvalidEscrowStatus.selector);
        escrow.emergencyRefund(ESCROW_ID);
    }

    // ============================================
    // 13. EDGE CASES AND SECURITY
    // ============================================

    function test_CannotReleaseAfterRefund() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(relayer);
        escrow.refund(ESCROW_ID);

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.InvalidEscrowStatus.selector);
        escrow.release(ESCROW_ID);
    }

    function test_CannotRefundAfterRelease() public {
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        vm.prank(relayer);
        escrow.release(ESCROW_ID);

        vm.prank(relayer);
        vm.expectRevert(IAgentEscrow.InvalidEscrowStatus.selector);
        escrow.refund(ESCROW_ID);
    }

    function test_MultipleEscrows() public {
        bytes32 escrowId2 = keccak256("task2");
        
        vm.startPrank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);
        escrow.deposit(escrowId2, poster, newAgent, AMOUNT, BID_HASH);
        vm.stopPrank();

        IAgentEscrow.Escrow memory e1 = escrow.getEscrow(ESCROW_ID);
        IAgentEscrow.Escrow memory e2 = escrow.getEscrow(escrowId2);

        assertEq(e1.agent, agent);
        assertEq(e2.agent, newAgent);
    }

    function test_GetEscrow_NonExistent() public view {
        IAgentEscrow.Escrow memory e = escrow.getEscrow(keccak256("nonexistent"));
        assertEq(e.poster, address(0));
        assertEq(uint8(e.status), uint8(IAgentEscrow.EscrowStatus.None));
    }

    function testFuzz_Split(uint256 agentAmount) public {
        vm.assume(agentAmount <= AMOUNT);
        
        vm.prank(relayer);
        escrow.deposit(ESCROW_ID, poster, agent, AMOUNT, BID_HASH);

        uint256 posterBalBefore = usdc.balanceOf(poster);
        uint256 agentBalBefore = usdc.balanceOf(agent);

        vm.prank(relayer);
        escrow.split(ESCROW_ID, agentAmount);

        uint256 posterAmount = AMOUNT - agentAmount;
        assertEq(usdc.balanceOf(poster), posterBalBefore + posterAmount);
        assertEq(usdc.balanceOf(agent), agentBalBefore + agentAmount);
    }
}
