-- +goose Up
-- The regular price never changes; a sale sets an optional reduced price next to it.
ALTER TABLE product_variants
    ADD COLUMN sale_price bigint CHECK (sale_price IS NULL OR (sale_price > 0 AND sale_price < price));

UPDATE product_variants
SET price = compare_at_price, sale_price = price, compare_at_price = NULL
WHERE compare_at_price IS NOT NULL;

ALTER TABLE product_variants DROP COLUMN compare_at_price;

-- +goose Down
ALTER TABLE product_variants
    ADD COLUMN compare_at_price bigint CHECK (compare_at_price IS NULL OR compare_at_price > price);

UPDATE product_variants
SET compare_at_price = price, price = sale_price, sale_price = NULL
WHERE sale_price IS NOT NULL;

ALTER TABLE product_variants DROP COLUMN sale_price;
