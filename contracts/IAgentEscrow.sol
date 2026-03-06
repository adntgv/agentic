// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IAgentEscrow {
    enum TaskState {
        None,           // 0 - doesn't exist
        Funded,         // 1 - poster deposited, waiting for agent assignment
        Active,         // 2 - agent assigned, working
        Submitted,      // 3 - agent submitted work
        Disputed,       // 4 - either party raised dispute
        Completed,      // 5 - funds released to agent
        Cancelled,      // 6 - funds returned to poster
        Resolved        // 7 - dispute resolved by arbitrator
    }

    struct Task {
        address poster;
        address agent;
        uint256 amount;          // USDC amount (6 decimals)
        uint256 platformFee;     // fee amount calculated at creation
        uint64  deadline;        // unix timestamp: auto-release if poster doesn't act
        uint64  submittedAt;     // when agent submitted work
        TaskState state;
    }

    event TaskCreated(bytes32 indexed taskId, address indexed poster, uint256 amount, uint64 deadline);
    event AgentAssigned(bytes32 indexed taskId, address indexed agent);
    event WorkSubmitted(bytes32 indexed taskId, address indexed agent, uint64 submittedAt);
    event TaskApproved(bytes32 indexed taskId, uint256 agentPayout, uint256 platformFee);
    event TaskCancelled(bytes32 indexed taskId, uint256 refundAmount);
    event DisputeRaised(bytes32 indexed taskId, address indexed raisedBy);
    event DisputeResolved(bytes32 indexed taskId, uint256 agentAmount, uint256 posterAmount);

    function createTask(bytes32 taskId, uint256 amount, uint64 deadline) external;
    function assignAgent(bytes32 taskId, address agent) external;
    function submitWork(bytes32 taskId) external;
    function approveWork(bytes32 taskId) external;
    function cancelTask(bytes32 taskId) external;
    function raiseDispute(bytes32 taskId) external;
    function resolveDispute(bytes32 taskId, uint256 agentBps) external;
    function claimAfterDeadline(bytes32 taskId) external;
}
