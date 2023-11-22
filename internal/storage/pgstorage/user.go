package pgstorage

import (
	"context"
	"errors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/superles/yapgofermart/internal/model"
)

func (s *PgStorage) GetUserByID(ctx context.Context, id int64) (model.User, error) {

	item := model.User{}

	row := s.db.QueryRow(ctx, `SELECT id, name, password_hash, role, coalesce(balance, 0) from users where id=$1`, id)

	if row == nil {
		return item, errors.New("объект row пустой")
	}

	if err := row.Scan(&item.ID, &item.Name, &item.PasswordHash, &item.Role, &item.Balance); err != nil {
		return item, err
	}

	return item, nil
}

func (s *PgStorage) GetUserByName(ctx context.Context, name string) (model.User, error) {

	item := model.User{}

	row := s.db.QueryRow(ctx, `SELECT id, name, password_hash, role, coalesce(balance, 0) from users where name=$1`, name)

	if row == nil {
		return item, errors.New("объект row пустой")
	}

	if err := row.Scan(&item.ID, &item.Name, &item.PasswordHash, &item.Role, &item.Balance); err != nil {
		return item, err
	}

	return item, nil
}

func (s *PgStorage) RegisterUser(ctx context.Context, data model.User) (model.User, error) {
	_, err := s.db.Exec(ctx, "insert into users (name, password_hash, role) VALUES ($1, $2, $3)", data.Name, data.PasswordHash, data.Role)
	if err != nil {
		return data, err
	}
	return s.GetUserByName(ctx, data.Name)
}
