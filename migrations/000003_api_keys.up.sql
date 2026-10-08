CREATE TABLE api_keys (
    id           bigserial    PRIMARY KEY,
    created_at   timestamptz  NOT NULL DEFAULT now(),
    updated_at   timestamptz  NOT NULL DEFAULT now(),
    is_active    boolean      NOT NULL DEFAULT true,
    user_id      bigint       NOT NULL REFERENCES users(id),
    name         varchar(255) NOT NULL,
    prefix       varchar(8)   NOT NULL,
    key_hash     varchar(255) NOT NULL,
    expires_at   timestamptz,
    last_used_at timestamptz
);

CREATE INDEX idx_api_keys_prefix ON api_keys (prefix) WHERE is_active = true;
CREATE INDEX idx_api_keys_user_id ON api_keys (user_id);
