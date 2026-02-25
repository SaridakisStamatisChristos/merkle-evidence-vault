package merklerpc

import (
	"errors"
)

// Client is a stubbed gRPC client implementing the domain merkle Engine.
type Client struct{}

func NewClient(target string) *Client { return &Client{} }

func (c *Client) AppendLeaf(leaf []byte) (int64, []byte, error) {
	return 0, nil, errors.New("not implemented")
}

func (c *Client) InclusionProof(leafIndex int64) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (c *Client) ConsistencyProof(oldSize, newSize int64) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (c *Client) TreeSize() (int64, error) { return 0, errors.New("not implemented") }

func (c *Client) Root() ([]byte, error) { return nil, errors.New("not implemented") }
