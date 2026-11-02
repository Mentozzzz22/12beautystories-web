-- +goose Up
-- Discounts for sets of products from one product line, e.g. "shampoo + treatment 200 ml → −15%".
CREATE TABLE bundle_offers (
    id        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name      text        NOT NULL,
    is_active boolean     NOT NULL DEFAULT true,
    starts_at timestamptz,
    ends_at   timestamptz
);

-- A role in a set ("shampoo") and the products that can fill it.
CREATE TABLE bundle_groups (
    offer_id bigint NOT NULL REFERENCES bundle_offers (id) ON DELETE CASCADE,
    position int    NOT NULL,
    name     text   NOT NULL,
    PRIMARY KEY (offer_id, position)
);

CREATE TABLE bundle_group_products (
    offer_id   bigint NOT NULL,
    position   int    NOT NULL,
    product_id bigint NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    PRIMARY KEY (offer_id, position, product_id),
    FOREIGN KEY (offer_id, position) REFERENCES bundle_groups (offer_id, position) ON DELETE CASCADE
);

CREATE INDEX bundle_group_products_product_idx ON bundle_group_products (product_id);

CREATE TABLE bundle_tiers (
    id       bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    offer_id bigint NOT NULL REFERENCES bundle_offers (id) ON DELETE CASCADE,
    percent  bigint NOT NULL CHECK (percent BETWEEN 1 AND 100),
    sort     int    NOT NULL DEFAULT 0
);

CREATE INDEX bundle_tiers_offer_idx ON bundle_tiers (offer_id);

-- What a tier requires from a group. An empty label accepts any variant.
CREATE TABLE bundle_tier_slots (
    tier_id        bigint NOT NULL REFERENCES bundle_tiers (id) ON DELETE CASCADE,
    group_position int    NOT NULL,
    label          text   NOT NULL DEFAULT '',
    PRIMARY KEY (tier_id, group_position)
);

-- Applied discounts with their names, so an order keeps explaining its total after offers change.
ALTER TABLE orders ADD COLUMN discounts jsonb NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE orders DROP COLUMN discounts;
DROP TABLE bundle_tier_slots;
DROP TABLE bundle_tiers;
DROP TABLE bundle_group_products;
DROP TABLE bundle_groups;
DROP TABLE bundle_offers;
