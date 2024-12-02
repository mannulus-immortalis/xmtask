package db

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/mannulus-immortalis/xmtask/internal/models"
)

type db struct {
	db *sql.DB
}

func New(connStr string) (*db, error) {
	dbConn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	err = dbConn.Ping()
	if err != nil {
		return nil, err
	}
	return &db{db: dbConn}, nil
}

func NewFromConn(dbConn *sql.DB) *db {
	return &db{db: dbConn}
}

func (c *db) CreateItem(ctx context.Context, i *models.ItemCreateRequest) (*uuid.UUID, error) {
	query := `INSERT INTO companies (name, description, employee_count, is_registered, legal_type)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id`

	var id uuid.UUID
	err := c.db.QueryRowContext(ctx, query, i.Name, i.Description, i.EmployeeCount, i.IsRegistered, i.Type).Scan(&id)
	if err != nil && errIsDuplicate(err) {
		return nil, models.ErrDuplicateName
	}
	return &id, err
}

func (c *db) UpdateItem(ctx context.Context, id uuid.UUID, i *models.ItemUpdateRequest) error {
	query := `UPDATE companies 
	SET
		name=COALESCE($2, name), 
		description=COALESCE($3, description), 
		employee_count=COALESCE($4, employee_count), 
		is_registered=COALESCE($5, is_registered), 
		legal_type=COALESCE($6, legal_type)
	WHERE id = $1`

	res, err := c.db.ExecContext(ctx, query, id.String(), i.Name, i.Description, i.EmployeeCount, i.IsRegistered, i.Type)
	if err != nil && errIsDuplicate(err) {
		return models.ErrDuplicateName
	}
	if err != nil {
		return err
	}
	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if cnt == 0 {
		return models.ErrNotFound
	}

	return err
}

func (c *db) DeleteItem(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM companies WHERE id = $1`

	res, err := c.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return err
	}
	cnt, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if cnt == 0 {
		return models.ErrNotFound
	}
	return nil
}

func (c *db) GetItem(ctx context.Context, id uuid.UUID) (*models.ItemResponse, error) {
	query := `SELECT id, name, description, employee_count, is_registered, legal_type FROM companies WHERE id = $1`

	var i models.ItemResponse
	err := c.db.QueryRowContext(ctx, query, id.String()).Scan(&i.ID, &i.Name, &i.Description, &i.EmployeeCount, &i.IsRegistered, &i.Type)
	if err == sql.ErrNoRows {
		return nil, models.ErrNotFound
	}
	return &i, err
}

func (c *db) Close() {
	c.db.Close()
}

func errIsDuplicate(err error) bool {
	if pgerr, ok := err.(*pq.Error); ok {
		return pgerr.Code == "23505"
	}
	return false
}
