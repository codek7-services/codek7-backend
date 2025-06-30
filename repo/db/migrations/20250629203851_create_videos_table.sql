-- +goose Up
-- +goose StatementBegin
CREATE TABLE videos (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE videos;
-- +goose StatementEnd
