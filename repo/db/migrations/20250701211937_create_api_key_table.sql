-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS api_keys (
  id TEXT PRIMARY KEY,
  token TEXT NOT NULL,
  user_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS api_keys;
-- +goose StatementEnd

