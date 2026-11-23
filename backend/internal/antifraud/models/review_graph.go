package models

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	GraphUser    = "USER"
	GraphAccount = "BANK_ACCOUNT"
	GraphDevice  = "DEVICE"
	GraphIP      = "IP_ADDRESS"

	MaxGraphEdges = 2000
)

var ErrGraphRootNotFound = errors.New("graph root not found")

type GraphNode struct {
	Type    string
	Value   string
	OwnerID string
}

func (n GraphNode) ID() string {
	switch n.Type {
	case GraphUser:
		return "user:" + n.Value
	case GraphAccount:
		return "account:" + n.Value
	case GraphDevice:
		return "device:" + n.Value
	case GraphIP:
		return "ip:" + n.Value
	default:
		return ""
	}
}

type GraphEdge struct {
	ID         string
	Type       string
	Source     GraphNode
	Target     GraphNode
	Properties json.RawMessage
}

type GraphWindow struct {
	From time.Time
	To   time.Time
}

type GraphQuery struct {
	CheckID int32
	Root    GraphNode
	Window  *GraphWindow
	Depth   int
	Limit   int
}

type ReviewGraph struct {
	Nodes     []GraphNode
	Edges     []GraphEdge
	Truncated bool
}
