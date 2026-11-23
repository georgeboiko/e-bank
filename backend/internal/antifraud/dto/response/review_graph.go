package response

import (
	"encoding/json"

	"e-bank/backend/internal/antifraud/models"
)

type Graph struct {
	Nodes     []GraphNode `json:"nodes"`
	Edges     []GraphEdge `json:"edges"`
	Truncated bool        `json:"truncated"`
}

type GraphNode struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Label      string         `json:"label"`
	Properties map[string]any `json:"properties"`
}

type GraphEdge struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Source     string          `json:"source"`
	Target     string          `json:"target"`
	Properties json.RawMessage `json:"properties"`
}

func NewGraph(graph models.ReviewGraph) Graph {
	result := Graph{
		Nodes:     make([]GraphNode, 0, len(graph.Nodes)),
		Edges:     make([]GraphEdge, 0, len(graph.Edges)),
		Truncated: graph.Truncated,
	}
	for _, node := range graph.Nodes {
		result.Nodes = append(result.Nodes, newGraphNode(node))
	}
	for _, edge := range graph.Edges {
		result.Edges = append(result.Edges, GraphEdge{
			ID:         edge.ID,
			Type:       edge.Type,
			Source:     edge.Source.ID(),
			Target:     edge.Target.ID(),
			Properties: edge.Properties,
		})
	}
	return result
}

func newGraphNode(node models.GraphNode) GraphNode {
	result := GraphNode{ID: node.ID(), Type: node.Type, Label: node.Value}
	switch node.Type {
	case models.GraphUser:
		result.Label = "Пользователь " + node.Value
		result.Properties = map[string]any{"userId": node.Value}
	case models.GraphAccount:
		result.Label = "Счёт " + node.Value
		result.Properties = map[string]any{"accountId": json.Number(node.Value), "ownerId": node.OwnerID}
	case models.GraphDevice:
		result.Properties = map[string]any{"deviceId": node.Value}
	case models.GraphIP:
		result.Properties = map[string]any{"ipAddress": node.Value}
	}
	return result
}
