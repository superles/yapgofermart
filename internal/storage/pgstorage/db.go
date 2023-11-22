package pgstorage

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgStorage struct {
	db *pgxpool.Pool
}

func NewStorage(dsn string) (*PgStorage, error) {
	ctx := context.Background()
	dbConfig, dbErr := pgxpool.ParseConfig(dsn)

	if dbErr != nil {
		return nil, dbErr
	}

	db, err := pgxpool.NewWithConfig(ctx, dbConfig)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	checkTables(ctx, db)

	return &PgStorage{db}, nil
}

func checkTables(ctx context.Context, db *pgxpool.Pool) {
	db.QueryRow(ctx, `create table if not exists withdrawals
(
    id              integer generated always as identity
        constraint balance_pk
            primary key,
    order_number    varchar(255)                           not null,
    user_id         integer                                not null,
    sum             double precision,
    processed_at    timestamp with time zone default now() not null
);`)

	db.QueryRow(ctx, `create table if not exists public.balance
(
    id              integer generated always as identity
        constraint balance_pk
            primary key,
    order_number    varchar(255)                           not null,
    user_id         integer                                not null,
    current_balance double precision,
    accrual         double precision,
    withdrawal        double precision,
    processed_at    timestamp with time zone default now() not null
);`)

	db.QueryRow(ctx, `create table if not exists public.users
(
    id            integer generated always as identity
        constraint users_pk
            primary key,
    name          varchar(50)  not null,
    password_hash varchar(255) not null,
    role          varchar(50),
    balance       double precision
);`)

	db.QueryRow(ctx, `create table if not exists orders
(
    number           varchar(255)                           not null
        constraint orders_pk
            primary key,
    status           varchar(50)                            not null,
    accrual          double precision,
    uploaded_at      timestamp with time zone default now() not null,
    accrual_check_at timestamp with time zone,
    accrual_status   varchar(50),
    user_id          integer                                not null
);`)

	db.QueryRow(ctx, `create or replace function check_and_insert_order(inputNumber varchar(255), inputStatus varchar(50), inputUserID int) returns integer
    language plpgsql
as
$$
DECLARE
    resultCode INT;
BEGIN
    IF EXISTS (SELECT 1 FROM orders WHERE number = inputNumber) THEN
        IF EXISTS (SELECT 1 FROM orders WHERE number = inputNumber AND user_id = inputUserID) THEN
            resultCode := 1; -- Если существует и пользователь совпадает
        ELSE
            resultCode := 2; -- Если существует и пользователь не совпадает
        END IF;
    ELSE
        INSERT INTO orders (number, status, user_id)
        VALUES (inputNumber, inputStatus, inputUserID);
        resultCode := 3; -- Если запись добавлена
    END IF;

    RETURN resultCode;
END;
$$;`)

	db.QueryRow(ctx, `CREATE OR REPLACE FUNCTION update_balance_and_users()
    RETURNS TRIGGER AS $$
DECLARE
    sum double precision;
BEGIN
    IF NEW.status = 'PROCESSED' AND NEW.accrual > 0 THEN
        -- Добавляем запись в таблицу balance
        select coalesce(balance, 0) into sum from users where id = NEW.user_id;
        sum := sum + NEW.accrual;
        INSERT INTO balance (order_number, user_id, accrual, current_balance)
        VALUES (NEW.number, NEW.user_id, NEW.accrual, sum);

        -- Обновляем таблицу users
        UPDATE users
        SET balance = sum
        WHERE id = NEW.user_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER orders_update_trigger
    AFTER UPDATE OF status, accrual OR INSERT ON orders
    FOR EACH ROW
EXECUTE FUNCTION update_balance_and_users();`)

	db.QueryRow(ctx, `CREATE OR REPLACE FUNCTION update_balance_and_users_form_withdrawals()
    RETURNS TRIGGER AS $$
DECLARE
    sum double precision;
BEGIN
    IF NEW.sum > 0 THEN
        -- Добавляем запись в таблицу balance
        select coalesce(balance, 0) into sum from users where id = NEW.user_id;
        sum := sum - NEW.sum;
        IF sum > 0 THEN
            INSERT INTO balance (order_number, user_id, withdrawal, current_balance)
            VALUES (NEW.order_number, NEW.user_id, NEW.sum, sum);

            -- Обновляем таблицу users
            UPDATE users
            SET balance = sum
            WHERE id = NEW.user_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER withdrawals_update_trigger
    AFTER INSERT ON withdrawals
    FOR EACH ROW
EXECUTE FUNCTION update_balance_and_users_form_withdrawals();`)
}
