CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE restaurants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE menu_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    restaurant_id UUID NOT NULL REFERENCES restaurants(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price BIGINT NOT NULL CHECK (price >= 0),
    available BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_menu_items_restaurant ON menu_items(restaurant_id);

CREATE TABLE carts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL UNIQUE,
    restaurant_id UUID REFERENCES restaurants(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cart_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id UUID NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
    menu_item_id UUID NOT NULL REFERENCES menu_items(id),
    quantity INT NOT NULL CHECK (quantity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(cart_id, menu_item_id)
);

CREATE TABLE orders (
    id UUID PRIMARY KEY,
    session_id UUID NOT NULL,
    restaurant_id UUID NOT NULL REFERENCES restaurants(id),
    status TEXT NOT NULL,
    total_price BIGINT NOT NULL CHECK (total_price >= 0),
    external_order_id TEXT,
    idempotency_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id UUID PRIMARY KEY,
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id UUID NOT NULL,
    name TEXT NOT NULL,
    price BIGINT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_orders_session ON orders(session_id);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
CREATE UNIQUE INDEX idx_orders_idempotency ON orders(session_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

INSERT INTO restaurants(name, description, address)
VALUES ('Pizza House', 'Пицца и паста', 'Нижний Новгород, Большая Покровская 10');

INSERT INTO menu_items(restaurant_id, name, description, price)
SELECT id, 'Маргарита', 'Томатный соус, моцарелла, базилик', 650 FROM restaurants WHERE name='Pizza House';

INSERT INTO menu_items(restaurant_id, name, description, price)
SELECT id, 'Пепперони', 'Томатный соус, моцарелла, пепперони', 750 FROM restaurants WHERE name='Pizza House';

INSERT INTO menu_items(restaurant_id, name, description, price)
SELECT id, 'Паста Карбонара', 'Паста, бекон, сливочный соус', 590 FROM restaurants WHERE name='Pizza House';
