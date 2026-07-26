-- +goose Up
-- +goose StatementBegin
CREATE TABLE staffs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    hospital_id UUID NOT NULL REFERENCES hospitals(id),
    username VARCHAR(100) NOT NULL,
    password_hashed VARCHAR(128) NOT NULL,
    firstname VARCHAR(100) NOT NULL,
    midname VARCHAR(100),
    lastname VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX staffs_hospital_id_username_idx ON staffs (hospital_id, username);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE staffs;
-- +goose StatementEnd
