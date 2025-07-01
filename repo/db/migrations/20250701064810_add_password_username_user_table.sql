-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
  ADD COLUMN password VARCHAR(255),
  ADD COLUMN email VARCHAR(255);

ALTER TABLE users
  ADD CONSTRAINT users_email_key UNIQUE (email);

ALTER TABLE users
  ALTER COLUMN password SET NOT NULL,
  ALTER COLUMN email SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
  DROP CONSTRAINT users_email_key;

ALTER TABLE users
  DROP COLUMN password,
  DROP COLUMN email;
-- +goose StatementEnd
