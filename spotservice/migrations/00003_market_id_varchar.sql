-- +goose Up
ALTER TABLE markets
    ALTER COLUMN id TYPE VARCHAR(64);

-- +goose Down
ALTER TABLE markets
    ALTER COLUMN id TYPE TEXT;
