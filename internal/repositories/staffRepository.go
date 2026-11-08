package repositories

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

var ErrInvalidInvitation = errors.New("invalid or expired invitation")
var ErrStaffEmailAlreadyExists = errors.New("staff email already exists")

type PendingStaffInvitation struct {
	ID      string
	StaffID string
	UserID  string
	Name    string
	Email   string
}

type StaffRepository struct {
	db *sql.DB
}

func NewStaffRepository(db *sql.DB) *StaffRepository {
	return &StaffRepository{db: db}
}

// ------------------- STAFF CRUD -------------------

func (r *StaffRepository) GetAll(ctx context.Context) ([]models.Staff, error) {
	query := `SELECT id, name, email, COALESCE(phone, '') FROM staff ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	staffList := make([]models.Staff, 0)
	staffIndexes := make(map[string]int)
	for rows.Next() {
		var s models.Staff
		if err := rows.Scan(&s.ID, &s.Name, &s.Email, &s.Phone); err != nil {
			_ = rows.Close()
			return nil, err
		}
		s.Services = make([]string, 0)
		staffIndexes[s.ID] = len(staffList)
		staffList = append(staffList, s)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	serviceRows, err := r.db.QueryContext(ctx, `SELECT staff_id, service_id FROM staff_services`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = serviceRows.Close() }()
	for serviceRows.Next() {
		var staffID, serviceID string
		if err := serviceRows.Scan(&staffID, &serviceID); err != nil {
			return nil, err
		}
		if idx, ok := staffIndexes[staffID]; ok {
			staffList[idx].Services = append(staffList[idx].Services, serviceID)
		}
	}
	if err := serviceRows.Err(); err != nil {
		return nil, err
	}
	return staffList, nil
}

func (r *StaffRepository) Create(ctx context.Context, staff *models.Staff) error {
	if staff.ID == "" {
		staff.ID = uuid.New().String()
	}

	query := `
    INSERT INTO staff (id, name, email, phone)
    VALUES ($1, $2, $3, $4)
`
	_, err := r.db.ExecContext(ctx, query, staff.ID, staff.Name, staff.Email, staff.Phone)
	return err
}

// CreateInvitedStaff creates the staff resource, its pending auth identity,
// and the first invitation hash in one transaction. The raw invitation token
// is never accepted by or stored in this repository method.
func (r *StaffRepository) CreateInvitedStaff(ctx context.Context, staff *models.Staff, tokenHash string, expiresAt time.Time) error {
	staff.Email = strings.ToLower(strings.TrimSpace(staff.Email))
	if staff.ID == "" {
		staff.ID = uuid.NewString()
	}
	userID := uuid.NewString()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	var exists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM users WHERE LOWER(email) = $1
			UNION ALL
			SELECT 1 FROM staff WHERE LOWER(email) = $1
		)
	`, staff.Email).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrStaffEmailAlreadyExists
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO users (id, username, email, password, role, deleted_at, delete_deadline, account_status)
		VALUES ($1, $2, $3, '', 'staff', NULL, NULL, 'PENDING_INVITATION')
	`, userID, staff.Name, staff.Email); err != nil {
		return mapStaffUniqueViolation(err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO staff (id, name, email, phone, user_id)
		VALUES ($1, $2, $3, $4, $5)
	`, staff.ID, staff.Name, staff.Email, staff.Phone, userID); err != nil {
		return mapStaffUniqueViolation(err)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO staff_invitations (id, staff_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.NewString(), staff.ID, tokenHash, expiresAt); err != nil {
		return err
	}

	return tx.Commit()
}

func mapStaffUniqueViolation(err error) error {
	var postgresError *pq.Error
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		return ErrStaffEmailAlreadyExists
	}
	return err
}

func (r *StaffRepository) GetByID(ctx context.Context, id string) (*models.Staff, error) {
	query := `SELECT id, name, email, COALESCE(phone, '') FROM staff WHERE id = $1`
	var s models.Staff
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Email, &s.Phone,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *StaffRepository) Update(ctx context.Context, id string, staff *models.Staff) error {
	query := `UPDATE staff SET name = $1, email = $2, phone = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, staff.Name, staff.Email, staff.Phone, id)
	return err
}

func (r *StaffRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM staff WHERE id = $1`, id)
	return err
}

// ------------------- SERVICES (MANY-TO-MANY) -------------------

func (r *StaffRepository) GetServiceIDs(ctx context.Context, staffID string) ([]string, error) {
	query := `SELECT service_id FROM staff_services WHERE staff_id = $1`
	rows, err := r.db.QueryContext(ctx, query, staffID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
func (r *StaffRepository) GetStaffByService(ctx context.Context, serviceID string) ([]models.Staff, error) {
	query := `
    SELECT s.id, s.name, s.email, COALESCE(s.phone, '') AS phone
    FROM staff s
    JOIN staff_services ss ON s.id = ss.staff_id
    WHERE ss.service_id = $1
`

	rows, err := r.db.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var staffList []models.Staff
	for rows.Next() {
		var s models.Staff
		if err := rows.Scan(&s.ID, &s.Name, &s.Email, &s.Phone); err != nil {
			return nil, err
		}
		staffList = append(staffList, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return staffList, nil
}

// AssignServices uses a Sync pattern: Delete existing associations and re-insert
func (r *StaffRepository) AssignServices(ctx context.Context, staffID string, serviceIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	// 1. Clear current assignments
	if _, err := tx.ExecContext(ctx, `DELETE FROM staff_services WHERE staff_id = $1`, staffID); err != nil {
		return err
	}

	// 2. Insert new ones
	for _, svcID := range serviceIDs {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO staff_services (id, staff_id, service_id)
            VALUES ($1, $2, $3)
        `, uuid.New().String(), staffID, svcID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ------------------- SCHEDULE -------------------

func (r *StaffRepository) SetSchedule(ctx context.Context, staffID string, entries []map[string]string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	if _, err := tx.ExecContext(ctx, `DELETE FROM staff_schedule WHERE staff_id = $1`, staffID); err != nil {
		return err
	}

	for _, entry := range entries {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO staff_schedule (id, staff_id, day_of_week, start_time, end_time)
            VALUES ($1, $2, $3, $4, $5)
        `, uuid.New().String(), staffID, entry["day"], entry["start"], entry["end"])
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
func (r *StaffRepository) GetSchedule(ctx context.Context, staffID string) ([]map[string]string, error) {
	query := `SELECT day_of_week, start_time, end_time FROM staff_schedule WHERE staff_id = $1`
	rows, err := r.db.QueryContext(ctx, query, staffID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var schedule []map[string]string
	for rows.Next() {
		var day, start, end string
		if err := rows.Scan(&day, &start, &end); err != nil {
			return nil, err
		}
		schedule = append(schedule, map[string]string{
			"day":   day,
			"start": start,
			"end":   end,
		})
	}
	return schedule, nil
}

// ------------------- HOLIDAYS -------------------

func (r *StaffRepository) AddHoliday(ctx context.Context, staffID, date, reason string) error {
	query := `INSERT INTO staff_holidays (id, staff_id, date, reason) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, uuid.New().String(), staffID, date, reason)
	return err
}

// ReplaceInvitation atomically revokes any outstanding invitations for the
// staff member and inserts a new invitation record. Callers must pass only a
// cryptographic token digest, never the raw token.
func (r *StaffRepository) ReplaceInvitation(ctx context.Context, staffID, tokenHash string, expiresAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	if _, err := tx.ExecContext(ctx, `
		UPDATE staff_invitations
		SET revoked_at = NOW()
		WHERE staff_id = $1 AND consumed_at IS NULL AND revoked_at IS NULL
	`, staffID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO staff_invitations (id, staff_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.NewString(), staffID, tokenHash, expiresAt); err != nil {
		return err
	}

	return tx.Commit()
}

// ReplacePendingInvitationByEmail rotates an invitation only for a pending
// staff identity. The caller receives delivery details only when an eligible
// record exists; HTTP handlers must keep that distinction private.
func (r *StaffRepository) ReplacePendingInvitationByEmail(ctx context.Context, email, tokenHash string, expiresAt time.Time) (*PendingStaffInvitation, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer rollbackTx(tx)

	invitation := &PendingStaffInvitation{}
	err = tx.QueryRowContext(ctx, `
		SELECT staff.id, users.id, staff.name, staff.email
		FROM staff
		JOIN users ON users.id = staff.user_id
		WHERE LOWER(staff.email) = $1
		  AND users.account_status = 'PENDING_INVITATION'
	`, strings.ToLower(strings.TrimSpace(email))).Scan(
		&invitation.StaffID,
		&invitation.UserID,
		&invitation.Name,
		&invitation.Email,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	// Lock invitation rows before the user row, matching acceptance lock order.
	openInvitations, err := tx.QueryContext(ctx, `
		SELECT id FROM staff_invitations
		WHERE staff_id = $1 AND consumed_at IS NULL AND revoked_at IS NULL
		ORDER BY id FOR UPDATE
	`, invitation.StaffID)
	if err != nil {
		return nil, false, err
	}
	for openInvitations.Next() {
		var id string
		if err := openInvitations.Scan(&id); err != nil {
			_ = openInvitations.Close()
			return nil, false, err
		}
	}
	if err := openInvitations.Err(); err != nil {
		_ = openInvitations.Close()
		return nil, false, err
	}
	if err := openInvitations.Close(); err != nil {
		return nil, false, err
	}

	var status string
	if err := tx.QueryRowContext(ctx, `
		SELECT account_status FROM users WHERE id = $1 FOR UPDATE
	`, invitation.UserID).Scan(&status); err != nil {
		return nil, false, err
	}
	if status != "PENDING_INVITATION" {
		return nil, false, nil
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE staff_invitations
		SET revoked_at = NOW()
		WHERE staff_id = $1 AND consumed_at IS NULL AND revoked_at IS NULL
	`, invitation.StaffID); err != nil {
		return nil, false, err
	}
	invitation.ID = uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO staff_invitations (id, staff_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, invitation.ID, invitation.StaffID, tokenHash, expiresAt); err != nil {
		return nil, false, err
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	return invitation, true, nil
}

// Refactored to return both for the availability service
func (r *StaffRepository) GetAvailabilityData(ctx context.Context, staffID string) ([]map[string]string, []string, error) {
	// 1. Reuse your GetSchedule logic
	schedule, err := r.GetSchedule(ctx, staffID)
	if err != nil {
		return nil, nil, err
	}

	// 2. Get Holiday Dates
	hRows, err := r.db.QueryContext(ctx, `SELECT date FROM staff_holidays WHERE staff_id = $1`, staffID)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = hRows.Close()
	}()

	var holidays []string
	for hRows.Next() {
		var date string
		if err := hRows.Scan(&date); err != nil {
			return nil, nil, err
		}
		holidays = append(holidays, date)
	}

	return schedule, holidays, nil
}

// ------------------- INVITATIONS -------------------

// ConsumeInvitation atomically activates the linked users identity and marks
// the matching invitation consumed. The caller must pass a token hash.
func (r *StaffRepository) ConsumeInvitation(ctx context.Context, tokenHash, passwordHash string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	var invitationID, userID string
	err = tx.QueryRowContext(ctx, `
		SELECT invitation.id, staff.user_id
		FROM staff_invitations AS invitation
		JOIN staff ON staff.id = invitation.staff_id
		JOIN users ON users.id = staff.user_id
		WHERE invitation.token_hash = $1
		  AND invitation.expires_at > NOW()
		  AND invitation.consumed_at IS NULL
		  AND invitation.revoked_at IS NULL
		  AND users.account_status = 'PENDING_INVITATION'
	`, tokenHash).Scan(&invitationID, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidInvitation
	}
	if err != nil {
		return err
	}
	var lockedInvitationID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM staff_invitations
		WHERE id = $1 AND token_hash = $2 AND expires_at > NOW()
		  AND consumed_at IS NULL AND revoked_at IS NULL
		FOR UPDATE
	`, invitationID, tokenHash).Scan(&lockedInvitationID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidInvitation
	}
	if err != nil {
		return err
	}
	var accountStatus string
	if err := tx.QueryRowContext(ctx, `
		SELECT account_status FROM users WHERE id = $1 FOR UPDATE
	`, userID).Scan(&accountStatus); err != nil {
		return err
	}
	if accountStatus != "PENDING_INVITATION" {
		return ErrInvalidInvitation
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE users
		SET password = $1, account_status = 'ACTIVE'
		WHERE id = $2 AND account_status = 'PENDING_INVITATION'
	`, passwordHash, userID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return ErrInvalidInvitation
	}

	result, err = tx.ExecContext(ctx, `
		UPDATE staff_invitations
		SET consumed_at = NOW()
		WHERE id = $1 AND consumed_at IS NULL AND revoked_at IS NULL AND expires_at > NOW()
	`, invitationID)
	if err != nil {
		return err
	}
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return ErrInvalidInvitation
	}

	return tx.Commit()
}
