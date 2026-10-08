CREATE TABLE roles (
    id         bigserial    PRIMARY KEY,
    created_at timestamptz  NOT NULL DEFAULT now(),
    updated_at timestamptz  NOT NULL DEFAULT now(),
    is_active  boolean      NOT NULL DEFAULT true,
    name       varchar(100) NOT NULL UNIQUE
);

CREATE TABLE permissions (
    id         bigserial    PRIMARY KEY,
    created_at timestamptz  NOT NULL DEFAULT now(),
    updated_at timestamptz  NOT NULL DEFAULT now(),
    is_active  boolean      NOT NULL DEFAULT true,
    resource   varchar(100) NOT NULL,
    action     varchar(100) NOT NULL,
    UNIQUE(resource, action)
);

CREATE TABLE role_permissions (
    role_id       bigint NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id bigint NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id bigint NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- Seed default roles
INSERT INTO roles (name) VALUES ('user'), ('admin');
