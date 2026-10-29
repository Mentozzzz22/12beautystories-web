-- +goose Up
-- One row per person, identified by the normalized phone. Order statistics are not stored here:
-- they are aggregated from orders so that cancellations never leave stale counters behind.
CREATE TABLE customers (
    id                   bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    phone                text        NOT NULL UNIQUE,
    name                 text        NOT NULL,
    note                 text        NOT NULL DEFAULT '',
    marketing_consent    boolean     NOT NULL DEFAULT false,
    -- When consent was last given or withdrawn; the proof required by personal data law.
    marketing_consent_at timestamptz,
    created_at           timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE orders
    ADD COLUMN customer_id         bigint REFERENCES customers (id),
    ADD COLUMN privacy_accepted_at timestamptz;

-- Existing orders: one customer per phone, named after their latest order.
INSERT INTO customers (phone, name, created_at)
SELECT phone, (array_agg(customer_name ORDER BY created_at DESC, id DESC))[1], min(created_at)
FROM orders
GROUP BY phone;

UPDATE orders o SET customer_id = c.id FROM customers c WHERE c.phone = o.phone;

ALTER TABLE orders ALTER COLUMN customer_id SET NOT NULL;

CREATE INDEX orders_customer_idx ON orders (customer_id);

-- +goose Down
DROP INDEX orders_customer_idx;
ALTER TABLE orders DROP COLUMN privacy_accepted_at, DROP COLUMN customer_id;
DROP TABLE customers;
