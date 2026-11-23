package repositories

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"time"

	"e-bank/backend/internal/antifraud/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReviewGraphRepository struct {
	pool *pgxpool.Pool
}

func NewReviewGraphRepository(pool *pgxpool.Pool) *ReviewGraphRepository {
	return &ReviewGraphRepository{pool: pool}
}

type ReviewGraphSnapshot struct {
	tx pgx.Tx
}

func (r *ReviewGraphRepository) WithSnapshot(ctx context.Context, read func(ReviewGraphSnapshot) error) error {
	return pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	}, func(tx pgx.Tx) error {
		return read(ReviewGraphSnapshot{tx: tx})
	})
}

func (s ReviewGraphSnapshot) CheckRoot(ctx context.Context, checkID int32) (models.GraphNode, int32, time.Time, error) {
	root := models.GraphNode{Type: models.GraphAccount}
	var transferID int32
	var occurredAt time.Time
	err := s.tx.QueryRow(ctx, `
		SELECT a.id::text, a.owner_id, t.id, t.created_at
		FROM antifraud_check c
		JOIN transfer_attempt t ON t.id = c.transfer_id
		JOIN bank_account a ON a.id = t.from_account_id
		WHERE c.id = $1
	`, checkID).Scan(&root.Value, &root.OwnerID, &transferID, &occurredAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.GraphNode{}, 0, time.Time{}, models.ErrGraphRootNotFound
	}
	return root, transferID, occurredAt, err
}

func (s ReviewGraphSnapshot) ExplicitRoot(ctx context.Context, root models.GraphNode) (models.GraphNode, error) {
	var query string
	switch root.Type {
	case models.GraphAccount:
		err := s.tx.QueryRow(ctx, `SELECT owner_id FROM bank_account WHERE id = $1::text::integer`,
			root.Value).Scan(&root.OwnerID)
		if errors.Is(err, pgx.ErrNoRows) {
			return models.GraphNode{}, models.ErrGraphRootNotFound
		}
		return root, err
	case models.GraphUser:
		query = `SELECT EXISTS (SELECT 1 FROM app_user WHERE id = $1)`
	case models.GraphDevice:
		query = `SELECT EXISTS (SELECT 1 FROM transfer_attempt WHERE device = $1)`
	case models.GraphIP:
		err := s.tx.QueryRow(ctx, `
			SELECT host($1::text::inet)
			WHERE EXISTS (SELECT 1 FROM transfer_attempt WHERE ip_address = $1::text::inet)
			   OR EXISTS (SELECT 1 FROM login_event WHERE ip_address = $1::text::inet)
		`, root.Value).Scan(&root.Value)
		if errors.Is(err, pgx.ErrNoRows) {
			return models.GraphNode{}, models.ErrGraphRootNotFound
		}
		return root, err
	default:
		return models.GraphNode{}, models.ErrGraphRootNotFound
	}

	var exists bool
	if err := s.tx.QueryRow(ctx, query, root.Value).Scan(&exists); err != nil {
		return models.GraphNode{}, err
	}
	if !exists {
		return models.GraphNode{}, models.ErrGraphRootNotFound
	}
	return root, nil
}

func (s ReviewGraphSnapshot) Edges(
	ctx context.Context,
	nodes []models.GraphNode,
	window models.GraphWindow,
	transferID int32,
	onlyBetweenNodes bool,
	connectingEdges []string,
) ([]models.GraphEdge, error) {
	var users, devices []string
	var accounts []int32
	var ips []netip.Addr
	for _, node := range nodes {
		switch node.Type {
		case models.GraphUser:
			users = append(users, node.Value)
		case models.GraphAccount:
			id, err := strconv.ParseInt(node.Value, 10, 32)
			if err != nil {
				return nil, err
			}
			accounts = append(accounts, int32(id))
		case models.GraphDevice:
			devices = append(devices, node.Value)
		case models.GraphIP:
			ip, err := netip.ParseAddr(node.Value)
			if err != nil {
				return nil, err
			}
			ips = append(ips, ip)
		}
	}

	rows, err := s.tx.Query(ctx, graphEdgesQuery, window.From, window.To,
		accounts, users, devices, ips, transferID, onlyBetweenNodes, models.MaxGraphEdges+1, connectingEdges)
	if err != nil {
		return nil, fmt.Errorf("query graph edges: %w", err)
	}
	edges, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.GraphEdge, error) {
		var edge models.GraphEdge
		err := row.Scan(&edge.ID, &edge.Type, &edge.Source.Type, &edge.Source.Value, &edge.Source.OwnerID,
			&edge.Target.Type, &edge.Target.Value, &edge.Target.OwnerID, &edge.Properties)
		return edge, err
	})
	if err != nil {
		return nil, fmt.Errorf("read graph edges: %w", err)
	}
	return edges, nil
}

// Explore one edge per endpoint pair during BFS so parallel transfers do not hide new vertices.
// The final pass retains parallel edges and requires their endpoints to be selected.
const graphEdgesQuery = `
	WITH attempts AS (
		SELECT t.id, t.from_account_id, t.to_account_id, t.amount, t.device, t.ip_address, t.created_at,
		       sender.owner_id AS sender_id, recipient.owner_id AS recipient_id
		FROM transfer_attempt t
		JOIN bank_account sender ON sender.id = t.from_account_id
		JOIN bank_account recipient ON recipient.id = t.to_account_id
		WHERE t.created_at >= $1 AND t.created_at < $2
		  AND (t.from_account_id = ANY($3::integer[]) OR t.to_account_id = ANY($3)
		       OR sender.owner_id = ANY($4::text[]) OR t.device = ANY($5::text[])
		       OR t.ip_address = ANY($6::inet[]))
	), edges AS (
		SELECT 'owns:' || a.id AS id, 'OWNS' AS type,
		       'USER' AS source_type, a.owner_id AS source_value, '' AS source_owner,
		       'BANK_ACCOUNT' AS target_type, a.id::text AS target_value, a.owner_id AS target_owner,
		       jsonb_build_object('ownerId', a.owner_id) AS properties, 1 AS priority
		FROM bank_account a
		WHERE (a.id = ANY($3) OR a.owner_id = ANY($4))
		  AND (NOT $8 OR (a.id = ANY($3) AND a.owner_id = ANY($4)))
		UNION ALL
		SELECT 'transfer:' || t.id, 'ATTEMPTED_TRANSFER_TO',
		       'BANK_ACCOUNT', t.from_account_id::text, t.sender_id,
		       'BANK_ACCOUNT', t.to_account_id::text, t.recipient_id,
		       jsonb_build_object('transferId', t.id, 'amount', t.amount::text, 'occurredAt', t.created_at,
		                          'riskScore', c.risk_score, 'automaticDecision', c.automatic_decision,
		                          'deviceId', t.device, 'ipAddress', host(t.ip_address)),
		       CASE WHEN t.id = $7 THEN 0 ELSE 2 END
		FROM attempts t
		LEFT JOIN antifraud_check c ON c.transfer_id = t.id
		WHERE (t.from_account_id = ANY($3) OR t.to_account_id = ANY($3))
		  AND (NOT $8 OR (t.from_account_id = ANY($3) AND t.to_account_id = ANY($3)))
		UNION ALL
		SELECT 'uses-device:' || length(t.sender_id) || ':' || t.sender_id || ':' || t.device, 'USES_DEVICE',
		       'USER', t.sender_id, '', 'DEVICE', t.device, '',
		       jsonb_build_object('firstSeenAt', min(t.created_at), 'lastSeenAt', max(t.created_at), 'useCount', count(*)), 3
		FROM attempts t
		WHERE t.device IS NOT NULL
		  AND (t.sender_id = ANY($4) OR t.device = ANY($5))
		  AND (NOT $8 OR (t.sender_id = ANY($4) AND t.device = ANY($5)))
		GROUP BY t.sender_id, t.device
		UNION ALL
		SELECT 'login-ip:' || length(l.user_id) || ':' || l.user_id || ':' || host(l.ip_address), 'LOGGED_IN_FROM',
		       'USER', l.user_id, '', 'IP_ADDRESS', host(l.ip_address), '',
		       jsonb_build_object('firstSeenAt', min(l.occurred_at), 'lastSeenAt', max(l.occurred_at), 'useCount', count(*)), 4
		FROM login_event l
		WHERE l.occurred_at >= $1 AND l.occurred_at < $2
		  AND (l.user_id = ANY($4) OR l.ip_address = ANY($6))
		  AND (NOT $8 OR (l.user_id = ANY($4) AND l.ip_address = ANY($6)))
		GROUP BY l.user_id, l.ip_address
		UNION ALL
		SELECT 'device-ip:' || length(t.device) || ':' || t.device || ':' || host(t.ip_address), 'USED_FROM',
		       'DEVICE', t.device, '', 'IP_ADDRESS', host(t.ip_address), '',
		       jsonb_build_object('firstSeenAt', min(t.created_at), 'lastSeenAt', max(t.created_at), 'useCount', count(*)), 5
		FROM attempts t
		WHERE t.device IS NOT NULL AND t.ip_address IS NOT NULL
		  AND (t.device = ANY($5) OR t.ip_address = ANY($6))
		  AND (NOT $8 OR (t.device = ANY($5) AND t.ip_address = ANY($6)))
		GROUP BY t.device, t.ip_address
	), candidates AS (
		SELECT * FROM edges WHERE $8
		UNION ALL
		(SELECT DISTINCT ON (source_type, source_value, target_type, target_value) *
		 FROM edges WHERE NOT $8
		 ORDER BY source_type, source_value, target_type, target_value, priority, id)
	)
	SELECT id, type, source_type, source_value, source_owner, target_type, target_value, target_owner, properties
	FROM candidates ORDER BY (id = ANY($10::text[])) DESC NULLS LAST, priority, id LIMIT $9
`
