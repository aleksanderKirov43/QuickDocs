package docs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

type DocumentRepository interface {
	Create(ctx context.Context, doc *Document) error
	Get(ctx context.Context, id uuid.UUID) (*Document, error)
	ListAll(ctx context.Context) ([]Document, error)
	List(ctx context.Context, limit, offset int) ([]*Document, error)
	ListForUser(ctx context.Context, userID, limit, offset int) ([]*Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByUser(ctx context.Context, userID int) ([]Document, error)
	GetDocumentByID(ctx context.Context, docID string) (*Document, error)
	ListForUserFiltered(ctx context.Context, userID int, key, value string, limit, offset int, sortBy, order string) ([]*Document, error)
	ListPublicByLogin(ctx context.Context, login string, key, value string, limit, offset int, sortBy, order string) ([]*Document, error)
}

func NewDocsRepository(db *sql.DB) DocumentRepository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, doc *Document) error {
	if doc.Created.IsZero() {
		doc.Created = time.Now()
	}

	query := `INSERT INTO documents (id, owner_id, name, mime, has_file, is_public, json_data, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.db.ExecContext(ctx, query,
		doc.ID, doc.OwnerID, doc.Name, doc.Mime,
		doc.File, doc.Public, doc.JsonData, doc.Created)

	return err
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Document, error) {
	query := `SELECT id, owner_id, name, mime, has_file, is_public, json_data, created_at FROM documents WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var d Document
	err := row.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.JsonData, &d.Created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &d, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM documents WHERE id = $1`, id)
	return err
}

func (r *Repository) ListAll(ctx context.Context) ([]Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, name, mime, has_file, is_public, json_data, created_at FROM documents ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.JsonData, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]*Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, name, mime, has_file, is_public, json_data, created_at FROM documents ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.JsonData, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, nil
}

func (r *Repository) ListForUser(ctx context.Context, userID int, limit, offset int) ([]*Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, name, mime, has_file, is_public, json_data, created_at FROM documents
		 WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.JsonData, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID int) ([]Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, name, mime, has_file, is_public, json_data, created_at FROM documents WHERE owner_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.JsonData, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (r *Repository) GetDocumentByID(ctx context.Context, docID string) (*Document, error) {
	query := `SELECT id, owner_id, name, mime, has_file, is_public, json_data, created_at FROM documents WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, docID)

	var d Document
	err := row.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.JsonData, &d.Created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &d, nil
}

func (r *Repository) ListForUserFiltered(ctx context.Context, userID int, key, value string, limit, offset int, sortBy, order string) ([]*Document, error) {
	where := []string{"owner_id = $1"}
	args := []interface{}{userID}

	// белый список ключей
	switch key {
	case "", "name", "mime", "has_file", "is_public":
		if key != "" && value != "" {
			where = append(where, fmt.Sprintf("%s = $%d", key, len(args)+1))
			args = append(args, value)
		}
	default:
		// игнорируем неизвестный ключ
	}

	// сортировка
	col := "created_at"
	if sortBy == "name" {
		col = "name"
	}
	ord := "DESC"
	if strings.ToUpper(order) == "ASC" {
		ord = "ASC"
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`SELECT id, owner_id, name, mime, has_file, is_public, json_data, created_at FROM documents
		WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), col, ord, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.JsonData, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, nil
}

func (r *Repository) ListPublicByLogin(ctx context.Context, login string, key, value string, limit, offset int, sortBy, order string) ([]*Document, error) {
	where := []string{"u.login = $1", "d.is_public = true"}
	args := []interface{}{login}

	switch key {
	case "", "name", "mime", "has_file":
		if key != "" && value != "" {
			where = append(where, fmt.Sprintf("d.%s = $%d", key, len(args)+1))
			args = append(args, value)
		}
	}

	col := "d.created_at"
	if sortBy == "name" {
		col = "d.name"
	}
	ord := "DESC"
	if strings.ToUpper(order) == "ASC" {
		ord = "ASC"
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`SELECT d.id, d.owner_id, d.name, d.mime, d.has_file, d.is_public, d.json_data, d.created_at
		FROM documents d JOIN users u ON d.owner_id = u.id
		WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), col, ord, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.JsonData, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, nil
}
