create table if not exists public.users
(
    id            integer generated always as identity
        constraint users_pk
            primary key,
    name          varchar(50)  not null,
    password_hash varchar(255) not null,
    role          varchar(50),
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