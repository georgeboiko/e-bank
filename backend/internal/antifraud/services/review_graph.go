package services

import (
	"context"
	"time"

	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"
)

type ReviewGraphService struct {
	repository *repositories.ReviewGraphRepository
}

func NewReviewGraphService(repository *repositories.ReviewGraphRepository) *ReviewGraphService {
	return &ReviewGraphService{repository: repository}
}

func (s *ReviewGraphService) Get(ctx context.Context, query models.GraphQuery) (models.ReviewGraph, error) {
	var graph models.ReviewGraph
	err := s.repository.WithSnapshot(ctx, func(snapshot repositories.ReviewGraphSnapshot) error {
		root, transferID, window, err := resolveGraphRoot(ctx, snapshot, query)
		if err != nil {
			return err
		}

		graph.Nodes = []models.GraphNode{root}
		seen := map[string]bool{root.ID(): true}
		frontier := graph.Nodes
		var connectingEdges []string
		for depth := 0; depth < query.Depth && len(frontier) != 0; depth++ {
			edges, err := snapshot.Edges(ctx, frontier, window, transferID, false, nil)
			if err != nil {
				return err
			}
			if len(edges) > models.MaxGraphEdges {
				graph.Truncated = true
			}
			next := len(graph.Nodes)
			for _, edge := range edges {
				for _, node := range []models.GraphNode{edge.Source, edge.Target} {
					id := node.ID()
					if seen[id] {
						continue
					}
					if len(graph.Nodes) == query.Limit {
						graph.Truncated = true
						break
					}
					seen[id] = true
					graph.Nodes = append(graph.Nodes, node)
					connectingEdges = append(connectingEdges, edge.ID)
				}
			}
			frontier = graph.Nodes[next:]
		}

		// Keep the edges that connect the chosen nodes even when parallel attempts fill the edge limit.
		edges, err := snapshot.Edges(ctx, graph.Nodes, window, transferID, true, connectingEdges)
		if err != nil {
			return err
		}
		if len(edges) > models.MaxGraphEdges {
			graph.Truncated = true
			edges = edges[:models.MaxGraphEdges]
		}
		graph.Edges = edges
		return nil
	})
	return graph, err
}

func resolveGraphRoot(
	ctx context.Context, snapshot repositories.ReviewGraphSnapshot, query models.GraphQuery,
) (models.GraphNode, int32, models.GraphWindow, error) {
	now := time.Now()
	var root models.GraphNode
	var transferID int32
	var err error
	window := models.GraphWindow{From: now.Add(-24 * time.Hour), To: now}
	if query.CheckID != 0 {
		var at time.Time
		root, transferID, at, err = snapshot.CheckRoot(ctx, query.CheckID)
		// PostgreSQL timestamps have microsecond precision. Use the next representable
		// instant as the exclusive upper bound so the triggering attempt is included.
		window = models.GraphWindow{From: at.Add(-time.Hour), To: at.Add(time.Microsecond)}
	} else {
		root, err = snapshot.ExplicitRoot(ctx, query.Root)
	}
	if query.Window != nil {
		window = *query.Window
	}
	// Round bounds up to the database precision to keep the interval inclusive-exclusive.
	const roundUp = time.Microsecond - time.Nanosecond
	window.From = window.From.Add(roundUp).Truncate(time.Microsecond)
	window.To = window.To.Add(roundUp).Truncate(time.Microsecond)
	return root, transferID, window, err
}
