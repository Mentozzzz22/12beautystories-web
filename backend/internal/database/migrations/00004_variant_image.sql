-- +goose Up
-- The gallery photo shown when a customer picks this variant; empty keeps the current photo.
ALTER TABLE product_variants ADD COLUMN image_url text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE product_variants DROP COLUMN image_url;
