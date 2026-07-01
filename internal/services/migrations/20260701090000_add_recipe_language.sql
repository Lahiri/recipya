-- +goose Up
ALTER TABLE recipes
    ADD COLUMN language TEXT NOT NULL DEFAULT 'en';

ALTER TABLE user_settings
    ADD COLUMN recipe_language TEXT NOT NULL DEFAULT 'auto';

-- +goose Down
ALTER TABLE user_settings
    DROP COLUMN recipe_language;

ALTER TABLE recipes
    DROP COLUMN language;
