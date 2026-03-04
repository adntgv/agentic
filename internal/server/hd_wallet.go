package server

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/sha3"
)

// HDDeriver derives per-user Ethereum addresses from a master private key
// using a BIP-32-like deterministic derivation scheme.
//
// Path: master_key + HMAC-SHA512(key=master_key, data="agentic-hd" || uint32(index))
// This gives each user a unique, deterministic deposit address derived from the platform key.
type HDDeriver struct {
	masterKey []byte // 32-byte master private key
}

// NewHDDeriver creates an HD deriver from a hex-encoded private key
func NewHDDeriver(hexKey string) (*HDDeriver, error) {
	hexKey = strings.TrimPrefix(hexKey, "0x")
	if len(hexKey) != 64 {
		return nil, fmt.Errorf("hd_wallet: private key must be 32 bytes (64 hex chars), got %d", len(hexKey))
	}

	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		b, err := hexByte(hexKey[i*2 : i*2+2])
		if err != nil {
			return nil, fmt.Errorf("hd_wallet: invalid hex in private key: %w", err)
		}
		key[i] = b
	}

	return &HDDeriver{masterKey: key}, nil
}

// DeriveAddress derives a unique Ethereum address for the given index
func (hd *HDDeriver) DeriveAddress(index int) (string, error) {
	childKey, err := hd.deriveChildKey(index)
	if err != nil {
		return "", err
	}

	// Get the public key
	curve := elliptic.P256() // We'll use secp256k1 params manually
	_ = curve

	// Use secp256k1 curve (Ethereum's curve)
	privKey, pubKey := secp256k1KeyFromBytes(childKey)
	if privKey == nil {
		return "", fmt.Errorf("hd_wallet: failed to derive valid key for index %d", index)
	}

	// Ethereum address = last 20 bytes of Keccak256(uncompressed_pubkey[1:])
	addr := pubKeyToAddress(pubKey)
	return addr, nil
}

// DerivePrivateKey returns the private key for a given index (for sweeping funds)
func (hd *HDDeriver) DerivePrivateKey(index int) ([]byte, error) {
	return hd.deriveChildKey(index)
}

// deriveChildKey uses HMAC-SHA512 to derive a child key from master + index
func (hd *HDDeriver) deriveChildKey(index int) ([]byte, error) {
	// HMAC-SHA512(key=masterKey, data="agentic-hd" || big-endian uint32(index))
	mac := hmac.New(sha512.New, hd.masterKey)
	mac.Write([]byte("agentic-hd"))
	indexBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(indexBytes, uint32(index))
	mac.Write(indexBytes)
	derived := mac.Sum(nil)

	// Use first 32 bytes as the child private key
	childKey := derived[:32]

	// Ensure the key is valid for secp256k1 (non-zero, less than curve order)
	keyInt := new(big.Int).SetBytes(childKey)
	if keyInt.Sign() == 0 {
		return nil, fmt.Errorf("hd_wallet: derived zero key for index %d", index)
	}
	// secp256k1 order
	order, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	if keyInt.Cmp(order) >= 0 {
		keyInt.Mod(keyInt, order)
		childKey = keyInt.Bytes()
		// Pad to 32 bytes
		if len(childKey) < 32 {
			padded := make([]byte, 32)
			copy(padded[32-len(childKey):], childKey)
			childKey = padded
		}
	}

	return childKey, nil
}

// secp256k1 curve parameters
var secp256k1Params = &elliptic.CurveParams{
	P:       bigFromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F"),
	N:       bigFromHex("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141"),
	B:       big.NewInt(7),
	Gx:      bigFromHex("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798"),
	Gy:      bigFromHex("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8"),
	BitSize: 256,
	Name:    "secp256k1",
}

func bigFromHex(s string) *big.Int {
	n, _ := new(big.Int).SetString(s, 16)
	return n
}

// secp256k1KeyFromBytes creates an ECDSA key pair from raw bytes using secp256k1
func secp256k1KeyFromBytes(keyBytes []byte) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	k := new(big.Int).SetBytes(keyBytes)
	if k.Sign() == 0 || k.Cmp(secp256k1Params.N) >= 0 {
		return nil, nil
	}

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: secp256k1Params,
		},
		D: k,
	}
	priv.PublicKey.X, priv.PublicKey.Y = secp256k1Params.ScalarBaseMult(keyBytes)
	if priv.PublicKey.X == nil {
		return nil, nil
	}
	return priv, &priv.PublicKey
}

// pubKeyToAddress computes Ethereum address from an ECDSA public key
func pubKeyToAddress(pub *ecdsa.PublicKey) string {
	// Uncompressed public key bytes (64 bytes: X || Y, without 0x04 prefix)
	xBytes := pub.X.Bytes()
	yBytes := pub.Y.Bytes()

	// Pad to 32 bytes each
	pubBytes := make([]byte, 64)
	copy(pubBytes[32-len(xBytes):32], xBytes)
	copy(pubBytes[64-len(yBytes):64], yBytes)

	// Keccak256
	hash := keccak256(pubBytes)

	// Last 20 bytes = address
	addr := hash[12:]
	return "0x" + fmt.Sprintf("%x", addr)
}

// keccak256 computes the Keccak-256 hash
func keccak256(data []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return h.Sum(nil)
}

// hexByte parses a 2-char hex string to a byte
func hexByte(s string) (byte, error) {
	var b byte
	for _, c := range s {
		b <<= 4
		switch {
		case c >= '0' && c <= '9':
			b |= byte(c - '0')
		case c >= 'a' && c <= 'f':
			b |= byte(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			b |= byte(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex char: %c", c)
		}
	}
	return b, nil
}


