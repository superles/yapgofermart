package pgstorage

import (
	"context"
	"fmt"
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

	err = checkTables(ctx, db)
	if err != nil {
		return nil, err
	}

	return &PgStorage{db}, nil
}

func checkTables(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
create table if not exists public.users
(
    name          varchar(50)  not null,
    password_hash varchar(255) not null,
    role          varchar(50),
    id            integer generated always as identity
        constraint users_pk
            primary key,
    balance       double precision
);

create table if not exists public.orders
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
);

create table if not exists public.balance
(
    id              integer generated always as identity
        constraint balance_pk
            primary key,
    order_number    varchar(255)                           not null,
    user_id         integer                                not null,
    current_balance double precision,
    accrual         double precision,
    withdrawal      double precision,
    processed_at    timestamp with time zone default now() not null
);

create table if not exists public.withdrawals
(
    id           integer generated always as identity
        constraint withdrawal_pk
            primary key,
    order_number varchar(255)                           not null,
    user_id      integer                                not null,
    sum          double precision,
    processed_at timestamp with time zone default now() not null
);

create or replace function public.check_and_insert_order(inputnumber character varying, inputstatus character varying,
                                                         inputuserid integer) returns integer
    language plpgsql
as
$$
DECLARE
    resultCode INT;
BEGIN
    IF EXISTS(SELECT 1 FROM orders WHERE number = inputNumber) THEN
        IF EXISTS(SELECT 1 FROM orders WHERE number = inputNumber AND user_id = inputUserID) THEN
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
$$;

create or replace function public.update_balance_and_users() returns trigger
    language plpgsql
as
$$
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
$$;

create or replace function public.update_balance_and_users_form_withdrawals() returns trigger
    language plpgsql
as
$$
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
$$;

create or replace trigger orders_update_trigger
    after insert or update
        of status, accrual
    on public.orders
    for each row
execute procedure public.update_balance_and_users();

create or replace trigger withdrawals_update_trigger
    after insert
    on public.withdrawals
    for each row
execute procedure public.update_balance_and_users_form_withdrawals();

`)
	if err != nil {
		return fmt.Errorf("update_balance_and_users create error: %w", err)
	}
	return nil
}
