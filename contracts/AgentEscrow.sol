// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "./IAgentEscrow.sol";

/**
 * @title AgentEscrow
 * @notice Escrow contract for AI agent marketplace. USDC on Base.
 * @dev No proxy pattern - simple, auditable, deploy-and-go.
 *      If upgrades needed, deploy v2 and migrate (tasks are independent).
 *
 * Flow: createTask → assignAgent → submitWork → approveWork → done
 * Disputes: raiseDispute → resolveDispute (arbitrator splits funds)
 * Auto-release: if poster doesn't act within `autoReleaseDays` after submission, agent can claim
 */
contract AgentEscrow is IAgentEscrow {
    // --- Storage ---
    IERC20 public immutable usdc;
    address public owner;
    address public arbitrator;
    address public feeRecipient;
    uint256 public feeBps;              // basis points, e.g. 500 = 5%
    uint256 public autoReleaseDays;     // days after submission before auto-release

    mapping(bytes32 => Task) public tasks;

    // --- Modifiers ---
    modifier onlyOwner() {
        require(msg.sender == owner, "not owner");
        _;
    }

    modifier onlyArbitrator() {
        require(msg.sender == arbitrator, "not arbitrator");
        _;
    }

    constructor(
        address _usdc,
        address _arbitrator,
        address _feeRecipient,
        uint256 _feeBps,
        uint256 _autoReleaseDays
    ) {
        require(_usdc != address(0), "zero usdc");
        require(_arbitrator != address(0), "zero arbitrator");
        require(_feeRecipient != address(0), "zero fee recipient");
        require(_feeBps <= 1000, "fee too high"); // max 10%
        require(_autoReleaseDays > 0, "zero auto-release");

        usdc = IERC20(_usdc);
        owner = msg.sender;
        arbitrator = _arbitrator;
        feeRecipient = _feeRecipient;
        feeBps = _feeBps;
        autoReleaseDays = _autoReleaseDays;
    }

    // --- Core Flow ---

    /// @notice Poster creates a task and deposits USDC into escrow
    /// @param taskId Unique task identifier (keccak256 of backend task ID)
    /// @param amount USDC amount (6 decimals)
    /// @param deadline Unix timestamp for task completion deadline
    function createTask(bytes32 taskId, uint256 amount, uint64 deadline) external {
        require(tasks[taskId].state == TaskState.None, "task exists");
        require(amount > 0, "zero amount");
        require(deadline > block.timestamp, "deadline passed");

        uint256 fee = (amount * feeBps) / 10000;

        tasks[taskId] = Task({
            poster: msg.sender,
            agent: address(0),
            amount: amount,
            platformFee: fee,
            deadline: deadline,
            submittedAt: 0,
            state: TaskState.Funded
        });

        require(usdc.transferFrom(msg.sender, address(this), amount), "transfer failed");

        emit TaskCreated(taskId, msg.sender, amount, deadline);
    }

    /// @notice Poster assigns an agent to the task
    function assignAgent(bytes32 taskId, address agent) external {
        Task storage t = tasks[taskId];
        require(t.state == TaskState.Funded, "not funded");
        require(msg.sender == t.poster, "not poster");
        require(agent != address(0), "zero agent");

        t.agent = agent;
        t.state = TaskState.Active;

        emit AgentAssigned(taskId, agent);
    }

    /// @notice Agent submits completed work
    function submitWork(bytes32 taskId) external {
        Task storage t = tasks[taskId];
        require(t.state == TaskState.Active, "not active");
        require(msg.sender == t.agent, "not agent");

        t.state = TaskState.Submitted;
        t.submittedAt = uint64(block.timestamp);

        emit WorkSubmitted(taskId, msg.sender, t.submittedAt);
    }

    /// @notice Poster approves work, funds released to agent minus platform fee
    function approveWork(bytes32 taskId) external {
        Task storage t = tasks[taskId];
        require(t.state == TaskState.Submitted, "not submitted");
        require(msg.sender == t.poster, "not poster");

        _releaseToAgent(taskId, t);
    }

    /// @notice Poster cancels task (only before agent assigned)
    function cancelTask(bytes32 taskId) external {
        Task storage t = tasks[taskId];
        require(t.state == TaskState.Funded, "cannot cancel");
        require(msg.sender == t.poster, "not poster");

        t.state = TaskState.Cancelled;

        require(usdc.transfer(t.poster, t.amount), "refund failed");

        emit TaskCancelled(taskId, t.amount);
    }

    /// @notice Either poster or agent can raise a dispute after work is submitted
    function raiseDispute(bytes32 taskId) external {
        Task storage t = tasks[taskId];
        require(
            t.state == TaskState.Submitted || t.state == TaskState.Active,
            "cannot dispute"
        );
        require(msg.sender == t.poster || msg.sender == t.agent, "not party");

        t.state = TaskState.Disputed;

        emit DisputeRaised(taskId, msg.sender);
    }

    /// @notice Arbitrator resolves dispute by splitting funds
    /// @param agentBps Basis points of (amount - fee) that go to agent. Rest to poster.
    function resolveDispute(bytes32 taskId, uint256 agentBps) external onlyArbitrator {
        Task storage t = tasks[taskId];
        require(t.state == TaskState.Disputed, "not disputed");
        require(agentBps <= 10000, "invalid bps");

        t.state = TaskState.Resolved;

        uint256 net = t.amount - t.platformFee;
        uint256 agentAmount = (net * agentBps) / 10000;
        uint256 posterAmount = net - agentAmount;

        // Platform always gets fee on disputes
        if (t.platformFee > 0) {
            require(usdc.transfer(feeRecipient, t.platformFee), "fee transfer failed");
        }
        if (agentAmount > 0) {
            require(usdc.transfer(t.agent, agentAmount), "agent transfer failed");
        }
        if (posterAmount > 0) {
            require(usdc.transfer(t.poster, posterAmount), "poster transfer failed");
        }

        emit DisputeResolved(taskId, agentAmount, posterAmount);
    }

    /// @notice Agent claims funds after auto-release period (poster unresponsive)
    function claimAfterDeadline(bytes32 taskId) external {
        Task storage t = tasks[taskId];
        require(t.state == TaskState.Submitted, "not submitted");
        require(msg.sender == t.agent, "not agent");
        require(
            block.timestamp >= t.submittedAt + (autoReleaseDays * 1 days),
            "too early"
        );

        _releaseToAgent(taskId, t);
    }

    // --- Internal ---

    function _releaseToAgent(bytes32 taskId, Task storage t) internal {
        t.state = TaskState.Completed;

        uint256 agentPayout = t.amount - t.platformFee;

        if (t.platformFee > 0) {
            require(usdc.transfer(feeRecipient, t.platformFee), "fee transfer failed");
        }
        require(usdc.transfer(t.agent, agentPayout), "agent transfer failed");

        emit TaskApproved(taskId, agentPayout, t.platformFee);
    }

    // --- Admin ---

    function setArbitrator(address _arbitrator) external onlyOwner {
        require(_arbitrator != address(0), "zero");
        arbitrator = _arbitrator;
    }

    function setFeeRecipient(address _feeRecipient) external onlyOwner {
        require(_feeRecipient != address(0), "zero");
        feeRecipient = _feeRecipient;
    }

    function setFeeBps(uint256 _feeBps) external onlyOwner {
        require(_feeBps <= 1000, "fee too high");
        feeBps = _feeBps;
    }

    function setAutoReleaseDays(uint256 _days) external onlyOwner {
        require(_days > 0, "zero");
        autoReleaseDays = _days;
    }

    function transferOwnership(address newOwner) external onlyOwner {
        require(newOwner != address(0), "zero");
        owner = newOwner;
    }

    // --- Views ---

    function getTask(bytes32 taskId) external view returns (Task memory) {
        return tasks[taskId];
    }
}

// Minimal ERC20 interface
interface IERC20 {
    function transferFrom(address from, address to, uint256 amount) external returns (bool);
    function transfer(address to, uint256 amount) external returns (bool);
    function approve(address spender, uint256 amount) external returns (bool);
    function balanceOf(address account) external view returns (uint256);
}
