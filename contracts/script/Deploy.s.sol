// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Script.sol";
import "../src/AgentEscrow.sol";

/**
 * @title DeployAgentEscrow
 * @notice Deployment script for AgentEscrow contract
 * @dev Run with: forge script script/Deploy.s.sol:DeployAgentEscrow --rpc-url <rpc> --broadcast
 */
contract DeployAgentEscrow is Script {
    // Base Mainnet USDC: 0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913
    // Base Sepolia USDC: Deploy MockUSDC for testing
    
    function run() external {
        // Load environment variables
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");
        address usdcAddress = vm.envAddress("USDC_ADDRESS");
        address relayerAddress = vm.envAddress("RELAYER_ADDRESS");
        address treasuryAddress = vm.envAddress("TREASURY_ADDRESS");

        vm.startBroadcast(deployerPrivateKey);

        // Deploy AgentEscrow
        AgentEscrow escrow = new AgentEscrow(
            usdcAddress,
            relayerAddress,
            treasuryAddress
        );

        console.log("AgentEscrow deployed at:", address(escrow));
        console.log("Admin:", escrow.admin());
        console.log("Relayer:", escrow.relayer());
        console.log("Treasury:", escrow.treasury());
        console.log("USDC:", address(escrow.usdc()));

        vm.stopBroadcast();
    }
}

/**
 * @title DeployMockUSDC
 * @notice Deploy MockUSDC for testing on testnets
 * @dev Run with: forge script script/Deploy.s.sol:DeployMockUSDC --rpc-url <rpc> --broadcast
 */
contract DeployMockUSDC is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("PRIVATE_KEY");

        vm.startBroadcast(deployerPrivateKey);

        // For deployment, we'll need to create a standalone MockUSDC in src/
        // For now, this references the test mock
        console.log("Deploy MockUSDC separately for testnet use");
        console.log("Use the MockUSDC from test/mocks/ as reference");

        vm.stopBroadcast();
    }
}
