-- +goose Up
CREATE TABLE brands (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug        text    NOT NULL UNIQUE,
    name        text    NOT NULL,
    description text    NOT NULL DEFAULT '',
    logo_url    text    NOT NULL DEFAULT '',
    sort        int     NOT NULL DEFAULT 0,
    is_active   boolean NOT NULL DEFAULT true
);

CREATE TABLE product_lines (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    brand_id    bigint  NOT NULL REFERENCES brands (id) ON DELETE CASCADE,
    slug        text    NOT NULL,
    name        text    NOT NULL,
    description text    NOT NULL DEFAULT '',
    image_url   text    NOT NULL DEFAULT '',
    sort        int     NOT NULL DEFAULT 0,
    is_active   boolean NOT NULL DEFAULT true,
    UNIQUE (brand_id, slug)
);

CREATE TABLE categories (
    id        bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug      text    NOT NULL UNIQUE,
    name      text    NOT NULL,
    sort      int     NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE products (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug              text        NOT NULL UNIQUE,
    brand_id          bigint      NOT NULL REFERENCES brands (id),
    line_id           bigint      REFERENCES product_lines (id) ON DELETE SET NULL,
    category_id       bigint      REFERENCES categories (id) ON DELETE SET NULL,
    name              text        NOT NULL,
    description       text        NOT NULL DEFAULT '',
    ingredients       text        NOT NULL DEFAULT '',
    -- What distinguishes variants on the product page: volume, colour or nothing.
    option_kind       text        NOT NULL DEFAULT 'none' CHECK (option_kind IN ('none', 'volume', 'color')),
    sort              int         NOT NULL DEFAULT 0,
    is_active         boolean     NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX products_brand_idx ON products (brand_id);
CREATE INDEX products_line_idx ON products (line_id);
CREATE INDEX products_category_idx ON products (category_id);

CREATE TABLE product_media (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id bigint NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    url        text   NOT NULL,
    kind       text   NOT NULL DEFAULT 'image' CHECK (kind IN ('image', 'video')),
    sort       int    NOT NULL DEFAULT 0
);

CREATE INDEX product_media_product_idx ON product_media (product_id);

CREATE TABLE product_variants (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id       bigint  NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    label            text    NOT NULL DEFAULT '',
    sku              text    NOT NULL DEFAULT '',
    price            bigint  NOT NULL CHECK (price >= 0),
    compare_at_price bigint  CHECK (compare_at_price IS NULL OR compare_at_price > price),
    stock            int     NOT NULL DEFAULT 0 CHECK (stock >= 0),
    sort             int     NOT NULL DEFAULT 0,
    is_active        boolean NOT NULL DEFAULT true
);

CREATE INDEX product_variants_product_idx ON product_variants (product_id);

CREATE TABLE banners (
    id               bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title            text        NOT NULL DEFAULT '',
    subtitle         text        NOT NULL DEFAULT '',
    image_url        text        NOT NULL,
    mobile_image_url text        NOT NULL DEFAULT '',
    -- Storefront path ("/brands/davines/energizing") or absolute URL.
    link_url         text        NOT NULL DEFAULT '',
    sort             int         NOT NULL DEFAULT 0,
    is_active        boolean     NOT NULL DEFAULT true,
    starts_at        timestamptz,
    ends_at          timestamptz
);

CREATE TABLE promo_codes (
    id                    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code                  text        NOT NULL UNIQUE,
    kind                  text        NOT NULL CHECK (kind IN ('percent', 'fixed')),
    value                 bigint      NOT NULL CHECK (value > 0),
    min_subtotal          bigint      NOT NULL DEFAULT 0,
    combinable_with_rules boolean     NOT NULL DEFAULT false,
    max_uses              int,
    used_count            int         NOT NULL DEFAULT 0,
    is_active             boolean     NOT NULL DEFAULT true,
    starts_at             timestamptz,
    ends_at               timestamptz,
    CHECK (kind <> 'percent' OR value <= 100)
);

-- Automatic cart discounts, e.g. "Davines from 50 000 ₸ → −10%".
CREATE TABLE discount_rules (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name         text        NOT NULL,
    brand_id     bigint      REFERENCES brands (id) ON DELETE CASCADE,
    min_subtotal bigint      NOT NULL DEFAULT 0,
    kind         text        NOT NULL CHECK (kind IN ('percent', 'fixed')),
    value        bigint      NOT NULL CHECK (value > 0),
    is_active    boolean     NOT NULL DEFAULT true,
    starts_at    timestamptz,
    ends_at      timestamptz,
    CHECK (kind <> 'percent' OR value <= 100)
);

CREATE TABLE orders (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    status          text        NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'awaiting_payment', 'paid', 'shipped', 'completed', 'cancelled')),
    customer_name   text        NOT NULL,
    phone           text        NOT NULL,
    delivery_method text        NOT NULL CHECK (delivery_method IN ('pickup', 'courier', 'post')),
    address         text        NOT NULL DEFAULT '',
    comment         text        NOT NULL DEFAULT '',
    promo_code      text        NOT NULL DEFAULT '',
    subtotal        bigint      NOT NULL,
    discount_total  bigint      NOT NULL DEFAULT 0,
    total           bigint      NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX orders_created_idx ON orders (created_at DESC);

CREATE TABLE order_items (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id      bigint NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    variant_id    bigint REFERENCES product_variants (id) ON DELETE SET NULL,
    -- Snapshot so the order stays readable after catalog edits.
    product_name  text   NOT NULL,
    variant_label text   NOT NULL DEFAULT '',
    unit_price    bigint NOT NULL,
    quantity      int    NOT NULL CHECK (quantity > 0),
    line_total    bigint NOT NULL
);

CREATE INDEX order_items_order_idx ON order_items (order_id);

-- +goose Down
DROP TABLE order_items;
DROP TABLE orders;
DROP TABLE discount_rules;
DROP TABLE promo_codes;
DROP TABLE banners;
DROP TABLE product_variants;
DROP TABLE product_media;
DROP TABLE products;
DROP TABLE categories;
DROP TABLE product_lines;
DROP TABLE brands;
