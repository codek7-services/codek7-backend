-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
