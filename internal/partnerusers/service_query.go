package partnerusers

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
)

// ListUsersByPartner lista usuarios de todas las tablas subscriber que
// correspondan al partner base solicitado.
func (s *Service) ListUsersByPartner(ctx context.Context, req ListUsersByPartnerRequest, authUser AuthUser) (ListUsersByPartnerResult, error) {
	if err := ensureAuthUser(authUser); err != nil {
		return ListUsersByPartnerResult{}, err
	}
	if strings.TrimSpace(req.Partner) == "" {
		return ListUsersByPartnerResult{}, fmt.Errorf("%w: partner is required", ErrValidation)
	}

	limit := req.Limit
	switch {
	case limit <= 0:
		limit = 100
	case limit > 500:
		limit = 500
	}

	tables, err := s.matchingSubscriberTablesForBase(ctx, req.Partner)
	if err != nil {
		return ListUsersByPartnerResult{}, err
	}
	if len(tables) == 0 {
		return ListUsersByPartnerResult{}, fmt.Errorf("%w: no subscriber tables found for partner %q", ErrNotFound, req.Partner)
	}

	users := make([]PartnerUserSummary, 0, limit)
	remaining := limit
	for _, table := range tables {
		if remaining <= 0 {
			break
		}

		rows, err := s.subscribers.ListByPartnerPrefix(ctx, table, s.basePartner(req.Partner)+"_", remaining)
		if err != nil {
			return ListUsersByPartnerResult{}, err
		}

		for _, row := range rows {
			users = append(users, PartnerUserSummary{
				Table:        table,
				SubscriberID: row.SubscriberID,
				PartnerID:    row.PartnerID,
				Email:        row.Email,
				Name:         row.Name,
				Status:       row.Status,
				Package:      row.Package,
			})
		}
		remaining = limit - len(users)
	}

	slices.SortFunc(users, func(a, b PartnerUserSummary) int {
		if result := cmp.Compare(strings.ToLower(b.PartnerID), strings.ToLower(a.PartnerID)); result != 0 {
			return result
		}
		return cmp.Compare(strings.ToLower(a.Table), strings.ToLower(b.Table))
	})

	return ListUsersByPartnerResult{
		Partner: s.basePartner(req.Partner),
		Total:   len(users),
		Users:   users,
	}, nil
}
