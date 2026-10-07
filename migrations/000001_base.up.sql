-- Items (example CRUD resource — delete when forking).
CREATE TABLE items (
    id              bigserial       PRIMARY KEY,
    created_at      timestamptz     NOT NULL DEFAULT now(),
    updated_at      timestamptz     NOT NULL DEFAULT now(),
    is_active       boolean         NOT NULL DEFAULT true,
    name            varchar(255)    NOT NULL,
    code            varchar(100)    NOT NULL UNIQUE,
    notes           jsonb
);

CREATE INDEX idx_items_active_id ON items (id) WHERE is_active = true;
