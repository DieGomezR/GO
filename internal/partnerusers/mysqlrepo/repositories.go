package mysqlrepo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"tienda-go/internal/partnerusers"
)

// UserRepository implementa el acceso a la tabla Users de la base app/mysql.
type UserRepository struct {
	db    *sql.DB
	table string
}

func (r *UserRepository) FindByID(ctx context.Context, userID int64) (*partnerusers.ManagedUser, error) {
	query := fmt.Sprintf("SELECT id, partnerId, email FROM %s WHERE id = ? LIMIT 1", quoteIdentifier(r.table))

	row := r.db.QueryRowContext(ctx, query, userID)
	var user partnerusers.ManagedUser
	if err := row.Scan(&user.ID, &user.PartnerID, &user.Email); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByPartnerID(ctx context.Context, partnerID string) (*partnerusers.ManagedUser, error) {
	query := fmt.Sprintf("SELECT id, partnerId, email FROM %s WHERE partnerId = ? LIMIT 1", quoteIdentifier(r.table))

	row := r.db.QueryRowContext(ctx, query, partnerID)
	var user partnerusers.ManagedUser
	if err := row.Scan(&user.ID, &user.PartnerID, &user.Email); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindPartnerIDByID(ctx context.Context, userID int64) (string, error) {
	query := fmt.Sprintf("SELECT partnerId FROM %s WHERE id = ? LIMIT 1", quoteIdentifier(r.table))

	var partnerID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, userID).Scan(&partnerID); err != nil {
		if err == sql.ErrNoRows {
			return "", partnerusers.ErrNotFound
		}
		return "", err
	}
	return partnerID.String, nil
}

func (r *UserRepository) FindPartnerIDByEmail(ctx context.Context, email string) (string, error) {
	query := fmt.Sprintf("SELECT partnerId FROM %s WHERE LOWER(email) = LOWER(?) LIMIT 1", quoteIdentifier(r.table))

	var partnerID sql.NullString
	if err := r.db.QueryRowContext(ctx, query, email).Scan(&partnerID); err != nil {
		if err == sql.ErrNoRows {
			return "", partnerusers.ErrNotFound
		}
		return "", err
	}
	return partnerID.String, nil
}

func (r *UserRepository) MaxPartnerSuffix(ctx context.Context, base string) (int, error) {
	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(CAST(SUBSTRING_INDEX(partnerId, '_', -1) AS UNSIGNED)), 0) FROM %s WHERE partnerId LIKE ? ESCAPE '\\\\'",
		quoteIdentifier(r.table),
	)

	pattern := escapeLike(base) + "\\_%"
	var max sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, pattern).Scan(&max); err != nil {
		return 0, err
	}
	return int(max.Int64), nil
}

func (r *UserRepository) SaveRegisteredUser(
	ctx context.Context,
	req partnerusers.RegisterRequest,
	partnerID, _ string,
	_ string,
	_ []partnerusers.ChannelActivation,
) error {
	// Sin el RegisterService original no conocemos todo el esquema final de Users.
	// Este port persiste el mínimo fiable para que la integración MySQL quede lista.
	query := fmt.Sprintf(
		"INSERT INTO %s (partnerId, email) VALUES (?, ?) ON DUPLICATE KEY UPDATE partnerId = VALUES(partnerId), email = VALUES(email)",
		quoteIdentifier(r.table),
	)
	_, err := r.db.ExecContext(ctx, query, partnerID, req.Email)
	return err
}

// ISPRepository implementa la búsqueda de partner -> país/territorio.
type ISPRepository struct {
	db    *sql.DB
	table string
}

func (r *ISPRepository) FindByPartnerID(ctx context.Context, partnerID string) (*partnerusers.ISP, error) {
	queryWithJSON := fmt.Sprintf(
		"SELECT partnerid, country, JSON_UNQUOTE(JSON_EXTRACT(config, '$.territory')) AS territory FROM %s WHERE partnerid = ? LIMIT 1",
		quoteIdentifier(r.table),
	)

	row := r.db.QueryRowContext(ctx, queryWithJSON, partnerID)
	var isp partnerusers.ISP
	var territory sql.NullString
	err := row.Scan(&isp.PartnerID, &isp.Country, &territory)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		// Fallback por si la tabla tiene un campo territory plano y no JSON config.
		querySimple := fmt.Sprintf(
			"SELECT partnerid, country, territory FROM %s WHERE partnerid = ? LIMIT 1",
			quoteIdentifier(r.table),
		)
		row = r.db.QueryRowContext(ctx, querySimple, partnerID)
		err = row.Scan(&isp.PartnerID, &isp.Country, &territory)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}

	isp.Territory = territory.String
	return &isp, nil
}

// SubscriberRepository implementa tablas dinámicas ISP_*_subscribers en db_prod.
type SubscriberRepository struct {
	db *sql.DB
}

func (r *SubscriberRepository) ExactTableExists(ctx context.Context, tableName string) (bool, error) {
	const query = `
		SELECT 1
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		LIMIT 1`

	var exists int
	err := r.db.QueryRowContext(ctx, query, tableName).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *SubscriberRepository) FindCaseInsensitiveTable(ctx context.Context, tableName string) (string, error) {
	const query = `
		SELECT TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE() AND LOWER(TABLE_NAME) = LOWER(?)
		LIMIT 1`

	var name sql.NullString
	if err := r.db.QueryRowContext(ctx, query, tableName).Scan(&name); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return name.String, nil
}

func (r *SubscriberRepository) ListSubscriberTables(ctx context.Context) ([]string, error) {
	const query = `
		SELECT TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		  AND (TABLE_NAME REGEXP '^(ISP_|isp_).+_subscribers$')`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := make([]string, 0, 32)
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func (r *SubscriberRepository) MaxPartnerSuffixOnTable(ctx context.Context, tableName, base string) (int, error) {
	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(CAST(SUBSTRING_INDEX(partnerId, '_', -1) AS UNSIGNED)), 0) FROM %s WHERE partnerId LIKE ? ESCAPE '\\\\'",
		quoteIdentifier(tableName),
	)

	pattern := escapeLike(base) + "\\_%"
	var max sql.NullInt64
	if err := r.db.QueryRowContext(ctx, query, pattern).Scan(&max); err != nil {
		return 0, err
	}
	return int(max.Int64), nil
}

func (r *SubscriberRepository) FindByPartnerID(ctx context.Context, tableName, partnerID string) (*partnerusers.SubscriberRecord, error) {
	query := fmt.Sprintf(subscriberSelectBase+" WHERE partnerId = ? LIMIT 1", quoteIdentifier(tableName))
	return r.querySubscriber(ctx, query, partnerID)
}

func (r *SubscriberRepository) FindBySubscriberID(ctx context.Context, tableName string, userID int64) (*partnerusers.SubscriberRecord, error) {
	query := fmt.Sprintf(subscriberSelectBase+" WHERE subscriber_id = ? LIMIT 1", quoteIdentifier(tableName))
	return r.querySubscriber(ctx, query, userID)
}

func (r *SubscriberRepository) FindByEmailAndStates(ctx context.Context, tableName, email string, states []string) (*partnerusers.SubscriberRecord, error) {
	if len(states) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(states)), ",")
	query := fmt.Sprintf(
		subscriberSelectBase+" WHERE LOWER(email) = LOWER(?) AND estado IN ("+placeholders+") LIMIT 1",
		quoteIdentifier(tableName),
	)

	args := make([]any, 0, len(states)+1)
	args = append(args, email)
	for _, state := range states {
		args = append(args, state)
	}
	return r.querySubscriber(ctx, query, args...)
}

func (r *SubscriberRepository) ListByPartnerPrefix(ctx context.Context, tableName, partnerPrefix string, limit int) ([]partnerusers.SubscriberRecord, error) {
	if limit <= 0 {
		limit = 100
	}

	query := fmt.Sprintf(
		subscriberSelectBase+" WHERE partnerId LIKE CONCAT(?, '%%') ORDER BY partnerId DESC LIMIT ?",
		quoteIdentifier(tableName),
	)

	rows, err := r.db.QueryContext(ctx, query, partnerPrefix, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]partnerusers.SubscriberRecord, 0, limit)
	for rows.Next() {
		record, err := scanSubscriberRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

func (r *SubscriberRepository) CreateMirroredUser(
	ctx context.Context,
	tableName string,
	req partnerusers.RegisterRequest,
	partnerID, readablePackage string,
	channels []partnerusers.ChannelActivation,
	externalID string,
	endDate *time.Time,
) error {
	query := fmt.Sprintf(`
		INSERT INTO %s
			(partnerId, subscriber_id, email, nombre, paquete, fecha_inicio, fecha_fin, departamento, ciudad, additional_channels_json, estado, ultima_actualizacion)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		quoteIdentifier(tableName),
	)

	now := time.Now().UTC()
	channelsJSON, err := marshalChannels(channels)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, query,
		partnerID,
		externalID,
		req.Email,
		req.Name,
		readablePackage,
		now,
		endDate,
		req.Department,
		req.City,
		channelsJSON,
		"activo",
		now,
	)
	return err
}

func (r *SubscriberRepository) UpdateByPartnerID(ctx context.Context, tableName, partnerID string, update partnerusers.SubscriberUpdate) (int64, error) {
	return r.updateWhere(ctx, tableName, "partnerId = ?", []any{partnerID}, update)
}

func (r *SubscriberRepository) UpdateBySubscriberID(ctx context.Context, tableName string, userID int64, update partnerusers.SubscriberUpdate) (int64, error) {
	return r.updateWhere(ctx, tableName, "subscriber_id = ?", []any{userID}, update)
}

func (r *SubscriberRepository) UpdateByEmailAndStates(ctx context.Context, tableName, email string, states []string, update partnerusers.SubscriberUpdate) (int64, error) {
	if len(states) == 0 {
		return 0, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(states)), ",")
	where := "LOWER(email) = LOWER(?) AND estado IN (" + placeholders + ")"
	args := make([]any, 0, len(states)+1)
	args = append(args, email)
	for _, state := range states {
		args = append(args, state)
	}
	return r.updateWhere(ctx, tableName, where, args, update)
}

func (r *SubscriberRepository) querySubscriber(ctx context.Context, query string, args ...any) (*partnerusers.SubscriberRecord, error) {
	row := r.db.QueryRowContext(ctx, query, args...)

	record, err := scanSubscriberRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (r *SubscriberRepository) updateWhere(
	ctx context.Context,
	tableName, where string,
	whereArgs []any,
	update partnerusers.SubscriberUpdate,
) (int64, error) {
	setClauses := make([]string, 0, 10)
	args := make([]any, 0, 10+len(whereArgs))

	if update.PartnerID != nil {
		setClauses = append(setClauses, "partnerId = ?")
		args = append(args, *update.PartnerID)
	}
	if update.SubscriberID != nil {
		setClauses = append(setClauses, "subscriber_id = ?")
		args = append(args, *update.SubscriberID)
	}
	if update.Status != nil {
		setClauses = append(setClauses, "estado = ?")
		args = append(args, *update.Status)
	}
	if update.Name != nil {
		setClauses = append(setClauses, "nombre = ?")
		args = append(args, *update.Name)
	}
	if update.Package != nil {
		setClauses = append(setClauses, "paquete = ?")
		args = append(args, *update.Package)
	}
	if update.StartDate != nil {
		setClauses = append(setClauses, "fecha_inicio = ?")
		args = append(args, *update.StartDate)
	}
	if update.EndDate != nil {
		setClauses = append(setClauses, "fecha_fin = ?")
		args = append(args, *update.EndDate)
	}
	if update.Department != nil {
		setClauses = append(setClauses, "departamento = ?")
		args = append(args, *update.Department)
	}
	if update.City != nil {
		setClauses = append(setClauses, "ciudad = ?")
		args = append(args, *update.City)
	}
	if update.AdditionalChannels != nil {
		channelsJSON, err := marshalChannels(update.AdditionalChannels)
		if err != nil {
			return 0, err
		}
		setClauses = append(setClauses, "additional_channels_json = ?")
		args = append(args, channelsJSON)
	}
	if update.RemovedAt != nil {
		setClauses = append(setClauses, "removed = ?")
		args = append(args, *update.RemovedAt)
	}
	if update.LastUpdatedAt != nil {
		setClauses = append(setClauses, "ultima_actualizacion = ?")
		args = append(args, *update.LastUpdatedAt)
	}

	if len(setClauses) == 0 {
		return 0, nil
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		quoteIdentifier(tableName),
		strings.Join(setClauses, ", "),
		where,
	)
	args = append(args, whereArgs...)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// AuditRepository implementa la inserción en early_deactivation_audits.
type AuditRepository struct {
	db    *sql.DB
	table string
}

func (r *AuditRepository) Insert(ctx context.Context, entry partnerusers.EarlyDeactivationAuditEntry) error {
	query := fmt.Sprintf(`
		INSERT INTO %s
			(target_user_id, target_partner_id, actor_id, actor_email, actor_role, country, timezone, reason, forced, allowed, rejection_reason, window_open, attempted_at_utc, attempted_at_local, meta, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		quoteIdentifier(r.table),
	)

	meta, err := json.Marshal(entry.Metadata)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	_, err = r.db.ExecContext(ctx, query,
		entry.TargetUserID,
		entry.TargetPartnerID,
		entry.ActorID,
		entry.ActorEmail,
		entry.ActorRole,
		entry.Country,
		entry.Timezone,
		entry.Reason,
		entry.Forced,
		entry.Allowed,
		entry.RejectionReason,
		entry.WindowOpen,
		entry.AttemptedAtUTC,
		entry.AttemptedAtLocal,
		string(meta),
		now,
		now,
	)
	return err
}

const subscriberSelectBase = `
	SELECT
		subscriber_id,
		partnerId,
		email,
		nombre,
		estado,
		paquete,
		fecha_inicio,
		fecha_fin,
		additional_channels_json,
		removed,
		ultima_actualizacion
	FROM %s`

func quoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	value = strings.ReplaceAll(value, `_`, `\_`)
	return value
}

func marshalChannels(channels []partnerusers.ChannelActivation) (string, error) {
	if channels == nil {
		return "", nil
	}
	raw, err := json.Marshal(channels)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func unmarshalChannels(raw string) ([]partnerusers.ChannelActivation, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var channels []partnerusers.ChannelActivation
	if err := json.Unmarshal([]byte(raw), &channels); err != nil {
		return nil, err
	}
	return channels, nil
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	parsed := value.Time
	return &parsed
}

func scanSubscriberRow(scanner interface{ Scan(dest ...any) error }) (partnerusers.SubscriberRecord, error) {
	var record partnerusers.SubscriberRecord
	var startDate, endDate, removedAt, updatedAt sql.NullTime
	var subscriberID sql.NullInt64
	var additionalJSON sql.NullString

	err := scanner.Scan(
		&subscriberID,
		&record.PartnerID,
		&record.Email,
		&record.Name,
		&record.Status,
		&record.Package,
		&startDate,
		&endDate,
		&additionalJSON,
		&removedAt,
		&updatedAt,
	)
	if err != nil {
		return partnerusers.SubscriberRecord{}, err
	}

	record.SubscriberID = subscriberID.Int64
	record.StartDate = nullTimePtr(startDate)
	record.EndDate = nullTimePtr(endDate)
	record.RemovedAt = nullTimePtr(removedAt)
	record.LastUpdatedAt = nullTimePtr(updatedAt)

	channels, err := unmarshalChannels(additionalJSON.String)
	if err != nil {
		return partnerusers.SubscriberRecord{}, err
	}
	record.AdditionalChannels = channels

	return record, nil
}
