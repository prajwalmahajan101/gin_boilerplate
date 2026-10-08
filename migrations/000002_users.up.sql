CREATE TABLE users (
    id              bigserial       PRIMARY KEY,
    created_at      timestamptz     NOT NULL DEFAULT now(),
    updated_at      timestamptz     NOT NULL DEFAULT now(),
    is_active       boolean         NOT NULL DEFAULT true,
    email           varchar(255)    NOT NULL UNIQUE,
    password_hash   varchar(255)    NOT NULL,
    role            varchar(50)     NOT NULL DEFAULT 'user',
    last_login_at   timestamptz
);

CREATE INDEX idx_users_email ON users (email) WHERE is_active = true;
