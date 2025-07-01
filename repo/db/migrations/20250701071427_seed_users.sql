-- +goose Up
-- +goose StatementBegin
INSERT INTO users (id, username, email, password, created_at)
VALUES
  (gen_random_uuid(), 'walid', 'walid@example.com', '123456', now()),
  (gen_random_uuid(), 'amira', 'amira@example.com', 'password', now()),
  (gen_random_uuid(), 'ahmed', 'ahmed@example.com', 'secret', now());
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM users
WHERE email IN ('walid@example.com', 'amira@example.com', 'ahmed@example.com');
-- +goose StatementEnd
