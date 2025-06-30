-- +goose Up
-- +goose StatementBegin
INSERT INTO users (id, username)
VALUES (gen_random_uuid(), 'walid');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM users WHERE username = 'walid';
-- +goose StatementEnd
