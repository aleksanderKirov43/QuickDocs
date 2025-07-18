package docs

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"time"
)

type Repository struct {
	db *sql.DB
}

type DocumentRepository interface {
	Create(ctx context.Context, doc *Document) error
	Get(ctx context.Context, id uuid.UUID) (*Document, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*Document, error)
	ListForUser(ctx context.Context, userID, limit, offset int) ([]*Document, error)
	ListAll(ctx context.Context) ([]Document, error)
	ListByUser(ctx context.Context, userID int) ([]Document, error)
	GetDocumentByID(ctx context.Context, docID string) (*Document, error)
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, doc *Document) error {
	if doc.Created.IsZero() {
		doc.Created = time.Now()
	}

	query := `INSERT INTO documents (id, owner_id, name, mime, has_file, is_public, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		doc.ID, doc.OwnerID, doc.Name, doc.Mime,
		doc.File, doc.Public, doc.Created)

	return err
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Document, error) {
	query := `SELECT id, owner_id, name, mime, has_file, is_public, created_at FROM documents WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)

	var d Document
	err := row.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.Created)
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
		`SELECT id, owner_id, name, mime, has_file, is_public, created_at FROM documents ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]*Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, name, mime, has_file, is_public, created_at FROM documents ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, nil
}

func (r *Repository) ListForUser(ctx context.Context, userID int, limit, offset int) ([]*Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, name, mime, has_file, is_public, created_at FROM documents
		 WHERE owner_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID int) ([]Document, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, owner_id, name, mime, has_file, is_public, created_at FROM documents WHERE owner_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.Created); err != nil {
			return nil, err
		}
		docs = append(docs, d)
	}
	return docs, nil
}

func (r *Repository) GetDocumentByID(ctx context.Context, docID string) (*Document, error) {
	query := `SELECT id, owner_id, name, mime, has_file, is_public, created_at FROM documents WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, docID)

	var d Document
	err := row.Scan(&d.ID, &d.OwnerID, &d.Name, &d.Mime, &d.File, &d.Public, &d.Created)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &d, nil
}
