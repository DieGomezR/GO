package partnerusers

import (
	"context"
	"testing"
	"time"
)

func TestListUsersByPartnerAggregatesAndSortsResults(t *testing.T) {
	t.Parallel()

	service := NewService(
		nil,
		nil,
		queryTestSubscribers{
			tables: []string{"ISP_TeleVVD_subscribers", "isp_televvd_subscribers"},
			rowsByTable: map[string][]SubscriberRecord{
				"ISP_TeleVVD_subscribers": {
					{PartnerID: "televvd_2", Email: "dos@example.com", Status: "activo"},
					{PartnerID: "televvd_1", Email: "uno@example.com", Status: "activo"},
				},
				"isp_televvd_subscribers": {
					{PartnerID: "televvd_3", Email: "tres@example.com", Status: "activo"},
				},
			},
		},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		"",
	)

	result, err := service.ListUsersByPartner(context.Background(), ListUsersByPartnerRequest{
		Partner: "TeleVVD",
		Limit:   10,
	}, AuthUser{PartnerID: "televvd", Role: "admin", Country: "co"})
	if err != nil {
		t.Fatalf("ListUsersByPartner returned error: %v", err)
	}

	if result.Partner != "televvd" {
		t.Fatalf("expected partner televvd, got %q", result.Partner)
	}
	if result.Total != 3 {
		t.Fatalf("expected total 3, got %d", result.Total)
	}

	expectedOrder := []string{"televvd_3", "televvd_2", "televvd_1"}
	for i, expected := range expectedOrder {
		if result.Users[i].PartnerID != expected {
			t.Fatalf("expected user %d to be %q, got %q", i, expected, result.Users[i].PartnerID)
		}
	}
}

type queryTestSubscribers struct {
	tables      []string
	rowsByTable map[string][]SubscriberRecord
}

func (q queryTestSubscribers) ExactTableExists(context.Context, string) (bool, error) {
	return false, nil
}

func (q queryTestSubscribers) FindCaseInsensitiveTable(context.Context, string) (string, error) {
	return "", nil
}

func (q queryTestSubscribers) ListSubscriberTables(context.Context) ([]string, error) {
	return append([]string(nil), q.tables...), nil
}

func (q queryTestSubscribers) MaxPartnerSuffixOnTable(context.Context, string, string) (int, error) {
	return 0, nil
}

func (q queryTestSubscribers) FindByPartnerID(context.Context, string, string) (*SubscriberRecord, error) {
	return nil, nil
}

func (q queryTestSubscribers) FindBySubscriberID(context.Context, string, int64) (*SubscriberRecord, error) {
	return nil, nil
}

func (q queryTestSubscribers) FindByEmailAndStates(context.Context, string, string, []string) (*SubscriberRecord, error) {
	return nil, nil
}

func (q queryTestSubscribers) ListByPartnerPrefix(_ context.Context, tableName, partnerPrefix string, limit int) ([]SubscriberRecord, error) {
	rows := make([]SubscriberRecord, 0, len(q.rowsByTable[tableName]))
	for _, row := range q.rowsByTable[tableName] {
		if len(rows) >= limit {
			break
		}
		if len(row.PartnerID) >= len(partnerPrefix) && row.PartnerID[:len(partnerPrefix)] == partnerPrefix {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (q queryTestSubscribers) CreateMirroredUser(context.Context, string, RegisterRequest, string, string, []ChannelActivation, string, *time.Time) error {
	return nil
}

func (q queryTestSubscribers) UpdateByPartnerID(context.Context, string, string, SubscriberUpdate) (int64, error) {
	return 0, nil
}

func (q queryTestSubscribers) UpdateBySubscriberID(context.Context, string, int64, SubscriberUpdate) (int64, error) {
	return 0, nil
}

func (q queryTestSubscribers) UpdateByEmailAndStates(context.Context, string, string, []string, SubscriberUpdate) (int64, error) {
	return 0, nil
}
