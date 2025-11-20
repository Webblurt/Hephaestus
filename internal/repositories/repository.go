package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	models "hephaestus/internal/models"
	utils "hephaestus/internal/utils"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB  *pgxpool.Pool
	log *utils.Logger
}

func NewRepository(cfg *utils.Config, log *utils.Logger) (*Repository, error) {
	tempRepo := &Repository{log: log}
	conn, err := tempRepo.CreateConnection(cfg)
	if err != nil {
		return nil, err
	}
	return &Repository{
		DB:  conn,
		log: log,
	}, nil
}

func (r *Repository) InsertTx(ctx context.Context, tx pgx.Tx, entity models.Entity) (string, error) {
	r.log.Debug("Inserting started...")
	columns, values, placeholders := buildInsertQuery(entity)

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id", entity.EntityName, columns, placeholders)

	r.log.Debug("Query execution: ", query)
	r.log.Debug("Values: ", values)
	var row pgx.Row
	if tx != nil {
		row = tx.QueryRow(ctx, query, values...)
	} else {
		row = r.DB.QueryRow(ctx, query, values...)
	}
	r.log.Debug("Query executed.")

	var id string
	if err := row.Scan(&id); err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) UpdateTx(ctx context.Context, tx pgx.Tx, entity models.Entity, id string) error {
	r.log.Debug("Updating started...")
	setClause, values := buildUpdateQuery(entity)
	if setClause == "" {
		return errors.New("no fields to update")
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", entity.EntityName, setClause, len(values)+1)
	values = append(values, id)

	r.log.Debug("Query execution: ", query)
	r.log.Debug("Values: ", values)
	_, err := tx.Exec(ctx, query, values...)
	if err != nil {
		return err
	}
	r.log.Debug("Query executed.")
	return nil
}

func (r *Repository) GetIDByNameTx(ctx context.Context, tx pgx.Tx, entity models.Entity) (string, error) {
	r.log.Debug("EntityName: '", entity.EntityName, "'")
	var column, value string
	for k, v := range entity.StringParameters {
		column = k
		value = v
		break
	}
	query := fmt.Sprintf("SELECT id FROM %s WHERE %s = $1", entity.EntityName, column)
	var id string
	r.log.Debug("Query execution: ", query)
	err := tx.QueryRow(ctx, query, value).Scan(&id)
	if err != nil {
		r.log.Debug("Error returned from query: ", err)
		if errors.Is(err, pgx.ErrNoRows) {
			r.log.Debug("No rows found, returning empty string + err 'no rows selected'")
			return "", errors.New("no rows selected")
		}
		return "", err
	}
	r.log.Debug("Query executed.")
	return id, nil
}

func buildInsertQuery(entity models.Entity) (columns string, values []interface{}, placeholders string) {
	var cols []string
	var ph []string
	var vals []interface{}

	i := 1
	for key, val := range entity.StringParameters {
		cols = append(cols, key)
		ph = append(ph, fmt.Sprintf("$%d", i))
		vals = append(vals, val)
		i++
	}
	for key, val := range entity.IntegerParameters {
		cols = append(cols, key)
		ph = append(ph, fmt.Sprintf("$%d", i))
		vals = append(vals, val)
		i++
	}
	for key, val := range entity.TimeParameters {
		cols = append(cols, key)
		ph = append(ph, fmt.Sprintf("$%d", i))
		vals = append(vals, val)
		i++
	}

	return strings.Join(cols, ", "), vals, strings.Join(ph, ", ")
}

func buildUpdateQuery(entity models.Entity) (setClause string, values []interface{}) {
	var setParts []string
	i := 1

	for key, val := range entity.StringParameters {
		if val == "" {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, i))
		values = append(values, val)
		i++
	}
	for key, val := range entity.IntegerParameters {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, i))
		values = append(values, val)
		i++
	}
	for key, val := range entity.TimeParameters {
		if val.IsZero() {
			continue
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", key, i))
		values = append(values, val)
		i++
	}
	if len(setParts) == 0 {
		return "", nil
	}

	return strings.Join(setParts, ", "), values
}

func NullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func NullIntToPtr(ni sql.NullInt64) *int64 {
	if ni.Valid {
		return &ni.Int64
	}
	return nil
}

func NullTimeToPtr(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
