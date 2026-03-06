package chain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// Client wraps an Ethereum RPC client with connection management and retry logic
type Client struct {
	rpcURL     string
	client     *ethclient.Client
	rpcClient  *rpc.Client
	maxRetries int
	retryDelay time.Duration
}

// NewClient creates a new Ethereum client with retry logic
func NewClient(rpcURL string, maxRetries int, retryDelay time.Duration) (*Client, error) {
	if rpcURL == "" {
		return nil, fmt.Errorf("rpc url cannot be empty")
	}

	rpcClient, err := rpc.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rpc: %w", err)
	}

	ethClient := ethclient.NewClient(rpcClient)

	return &Client{
		rpcURL:     rpcURL,
		client:     ethClient,
		rpcClient:  rpcClient,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}, nil
}

// ChainID returns the chain ID
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	return c.retryOperation(ctx, func() (*big.Int, error) {
		return c.client.ChainID(ctx)
	})
}

// BlockNumber returns the latest block number
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	var lastErr error
	for i := 0; i < c.maxRetries; i++ {
		num, err := c.client.BlockNumber(ctx)
		if err == nil {
			return num, nil
		}
		lastErr = err

		if i < c.maxRetries-1 {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(1<<uint(i))):
				// Exponential backoff
			}
		}
	}
	return 0, fmt.Errorf("block number query failed after %d retries: %w", c.maxRetries, lastErr)
}

// retryOperation retries a function with exponential backoff
func (c *Client) retryOperation(ctx context.Context, fn func() (*big.Int, error)) (*big.Int, error) {
	var lastErr error
	for i := 0; i < c.maxRetries; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err

		if i < c.maxRetries-1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(1<<uint(i))):
				// Exponential backoff
			}
		}
	}
	return nil, fmt.Errorf("operation failed after %d retries: %w", c.maxRetries, lastErr)
}

// Client returns the underlying ethclient
func (c *Client) Client() *ethclient.Client {
	return c.client
}

// RPC returns the underlying rpc.Client for low-level operations
func (c *Client) RPC() *rpc.Client {
	return c.rpcClient
}

// Close closes the client connection
func (c *Client) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected(ctx context.Context) bool {
	_, err := c.client.ChainID(ctx)
	return err == nil
}

// Reconnect attempts to reconnect to the RPC endpoint
func (c *Client) Reconnect(ctx context.Context) error {
	c.Close()

	rpcClient, err := rpc.DialContext(ctx, c.rpcURL)
	if err != nil {
		return fmt.Errorf("failed to reconnect to rpc: %w", err)
	}

	c.rpcClient = rpcClient
	c.client = ethclient.NewClient(rpcClient)

	return nil
}
