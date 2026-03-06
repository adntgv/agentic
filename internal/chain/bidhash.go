package chain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"golang.org/x/crypto/sha3"
	"math/big"
)

// TaskIDFromUUID converts a UUID to bytes32 via keccak256(abi.encodePacked(uuid_bytes16))
// Uses RFC 4122 Section 4.1.2 binary representation (network/big-endian byte order)
func TaskIDFromUUID(taskUUID uuid.UUID) [32]byte {
	// UUID is already 16 bytes in RFC 4122 format
	return crypto.Keccak256Hash(taskUUID[:])
}

// BidIDFromUUID converts a UUID to bytes32 (same as TaskIDFromUUID)
func BidIDFromUUID(bidUUID uuid.UUID) [32]byte {
	return crypto.Keccak256Hash(bidUUID[:])
}

// ComputeBidHash computes keccak256(abi.encode(domainSeparator, chainId, contract, taskId, agent, amount, etaHours, createdAt, bidId))
func ComputeBidHash(
	chainID uint64,
	contractAddr common.Address,
	taskID [32]byte,
	agent common.Address,
	amount *big.Int,
	etaHours uint32,
	createdAt uint64,
	bidID [32]byte,
) [32]byte {
	// Domain separator
	domainSeparator := "AI_AGENT_MARKETPLACE_V0_1"

	// ABI encode all fields
	// For proper ABI encoding:
	// - strings are dynamic, so they get (offset, length, data) encoding
	// - uint256, address, bytes32 are fixed-size and encoded directly
	
	// Create a hasher
	hasher := sha3.NewLegacyKeccak256()

	// Write domain separator (string - need to encode properly)
	// ABI encoding of string: keccak256(string_bytes)
	domainBytes := []byte(domainSeparator)
	hasher.Write(common.LeftPadBytes([]byte{byte(len(domainBytes))}, 32))
	hasher.Write(common.RightPadBytes(domainBytes, ((len(domainBytes)+31)/32)*32))

	// chainId (uint256)
	chainIDBig := new(big.Int).SetUint64(chainID)
	hasher.Write(common.LeftPadBytes(chainIDBig.Bytes(), 32))

	// contract address (address - 20 bytes, left-padded to 32)
	hasher.Write(common.LeftPadBytes(contractAddr.Bytes(), 32))

	// taskId (bytes32)
	hasher.Write(taskID[:])

	// agent (address)
	hasher.Write(common.LeftPadBytes(agent.Bytes(), 32))

	// amount (uint96 - padded to 32 bytes)
	hasher.Write(common.LeftPadBytes(amount.Bytes(), 32))

	// etaHours (uint32 - padded to 32 bytes)
	etaHoursBig := new(big.Int).SetUint64(uint64(etaHours))
	hasher.Write(common.LeftPadBytes(etaHoursBig.Bytes(), 32))

	// createdAt (uint40 - padded to 32 bytes)
	createdAtBig := new(big.Int).SetUint64(createdAt)
	hasher.Write(common.LeftPadBytes(createdAtBig.Bytes(), 32))

	// bidId (bytes32)
	hasher.Write(bidID[:])

	var result [32]byte
	copy(result[:], hasher.Sum(nil))
	return result
}

// VerifyBidHash verifies that a computed bidHash matches the expected hash
func VerifyBidHash(
	chainID uint64,
	contractAddr common.Address,
	taskID [32]byte,
	agent common.Address,
	amount *big.Int,
	etaHours uint32,
	createdAt uint64,
	bidID [32]byte,
	expectedHash [32]byte,
) bool {
	computed := ComputeBidHash(chainID, contractAddr, taskID, agent, amount, etaHours, createdAt, bidID)
	return computed == expectedHash
}
