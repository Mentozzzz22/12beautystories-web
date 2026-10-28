-- Demo data for local development. Run after the server applied migrations:
--   docker compose exec -T db psql -U shop -d shop < backend/db/seed.sql
BEGIN;

INSERT INTO brands (slug, name, sort) VALUES
    ('davines', 'Davines', 1),
    ('payot', 'Payot', 2);

INSERT INTO product_lines (brand_id, slug, name, sort)
SELECT b.id, l.slug, l.name, l.sort
FROM (VALUES
    ('davines', 'energizing', 'Energizing', 1),
    ('davines', 'minu', 'MINU', 2),
    ('payot', 'hydra-24', 'Hydra 24+', 1)
) AS l(brand, slug, name, sort)
JOIN brands b ON b.slug = l.brand;

INSERT INTO categories (slug, name, sort) VALUES
    ('shampoo', 'Шампуни', 1),
    ('conditioner', 'Кондиционеры', 2),
    ('leave-in', 'Несмываемые средства', 3),
    ('cream', 'Кремы', 4),
    ('accessories', 'Аксессуары', 5);

INSERT INTO products (slug, brand_id, line_id, category_id, name, description, ingredients, option_kind)
SELECT p.slug, b.id, l.id, c.id, p.name, p.description, p.ingredients, p.option_kind
FROM (VALUES
    ('davines-energizing-shampoo', 'davines', 'energizing', 'shampoo', 'Energizing Shampoo',
     'Энергетический шампунь для истончённых и склонных к выпадению волос.', 'Кофеин, ниацинамид', 'volume'),
    ('davines-minu-shampoo', 'davines', 'minu', 'shampoo', 'MINU Shampoo',
     'Шампунь для защиты цвета окрашенных волос.', 'Экстракт каперсов', 'volume'),
    ('payot-hydra-24-cream', 'payot', 'hydra-24', 'cream', 'Hydra 24+ Crème Glacée',
     'Увлажняющий крем-гель для обезвоженной кожи.', 'Гиалуроновая кислота', 'none'),
    ('davines-paddle-brush', 'davines', NULL, 'accessories', 'Paddle Brush',
     'Массажная расчёска.', '', 'color')
) AS p(slug, brand, line, category, name, description, ingredients, option_kind)
JOIN brands b ON b.slug = p.brand
LEFT JOIN product_lines l ON l.brand_id = b.id AND l.slug = p.line
LEFT JOIN categories c ON c.slug = p.category;

INSERT INTO product_variants (product_id, label, price, compare_at_price, stock, sort)
SELECT p.id, v.label, v.price, v.compare_at, v.stock, v.sort
FROM (VALUES
    ('davines-energizing-shampoo', '100 ml', 9000, NULL::bigint, 10, 1),
    ('davines-energizing-shampoo', '250 ml', 18000, NULL, 8, 2),
    ('davines-energizing-shampoo', '1000 ml', 48000, 54000, 2, 3),
    ('davines-minu-shampoo', '250 ml', 19500, NULL, 5, 1),
    ('payot-hydra-24-cream', '', 24000, NULL, 6, 1),
    ('davines-paddle-brush', 'Чёрная', 12000, NULL, 3, 1),
    ('davines-paddle-brush', 'Розовая', 12000, NULL, 0, 2)
) AS v(product, label, price, compare_at, stock, sort)
JOIN products p ON p.slug = v.product;

INSERT INTO product_media (product_id, url, sort)
SELECT p.id, 'https://placehold.co/800x800?text=' || replace(p.name, ' ', '+'), 0
FROM products p;

INSERT INTO banners (title, subtitle, image_url, link_url, sort) VALUES
    ('Davines Energizing', 'Новая линия против выпадения', 'https://placehold.co/1600x600?text=Energizing', '/brands/davines/energizing', 1),
    ('−10% на Davines', 'При заказе от 50 000 ₸', 'https://placehold.co/1600x600?text=Davines+-10%25', '/brands/davines', 2),
    ('Payot Hydra 24+', 'Увлажнение на весь день', 'https://placehold.co/1600x600?text=Payot', '/brands/payot/hydra-24', 3);

INSERT INTO discount_rules (name, brand_id, min_subtotal, kind, value)
SELECT 'Davines от 50 000 ₸: −10%', id, 50000, 'percent', 10 FROM brands WHERE slug = 'davines';

INSERT INTO promo_codes (code, kind, value, min_subtotal, combinable_with_rules) VALUES
    ('WELCOME5', 'percent', 5, 0, false);

COMMIT;
