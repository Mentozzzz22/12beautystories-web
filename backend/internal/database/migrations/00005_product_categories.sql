-- +goose Up
-- A product can be listed in several categories, e.g. an oil in both oils and leave-in care.
CREATE TABLE product_categories (
    product_id  bigint NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    category_id bigint NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, category_id)
);

CREATE INDEX product_categories_category_idx ON product_categories (category_id);

INSERT INTO product_categories (product_id, category_id)
SELECT id, category_id FROM products WHERE category_id IS NOT NULL;

DROP INDEX products_category_idx;
ALTER TABLE products DROP COLUMN category_id;

-- +goose Down
ALTER TABLE products ADD COLUMN category_id bigint REFERENCES categories (id) ON DELETE SET NULL;
CREATE INDEX products_category_idx ON products (category_id);

-- Only one category per product fits the old column; the lowest ID wins.
UPDATE products p
SET category_id = (SELECT min(pc.category_id) FROM product_categories pc WHERE pc.product_id = p.id);

DROP TABLE product_categories;
