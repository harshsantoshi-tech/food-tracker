CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    whatsapp_number TEXT NOT NULL UNIQUE,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE food_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    original_message TEXT NOT NULL,
    total_calories NUMERIC(8, 2) NOT NULL CHECK (total_calories >= 0),
    total_protein NUMERIC(8, 2) NOT NULL CHECK (total_protein >= 0),
    total_carbs NUMERIC(8, 2) NOT NULL CHECK (total_carbs >= 0),
    total_fat NUMERIC(8, 2) NOT NULL CHECK (total_fat >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_food_logs_user_id_created_at ON food_logs (user_id, created_at DESC);

CREATE TABLE food_items (
    id BIGSERIAL PRIMARY KEY,
    food_log_id BIGINT NOT NULL REFERENCES food_logs(id) ON DELETE CASCADE,
    food_name TEXT NOT NULL,
    quantity NUMERIC(8, 2) NOT NULL CHECK (quantity > 0),
    unit TEXT NOT NULL,
    estimated_weight_g NUMERIC(8, 2) NOT NULL CHECK (estimated_weight_g > 0),
    calories NUMERIC(8, 2) NOT NULL CHECK (calories >= 0),
    protein NUMERIC(8, 2) NOT NULL CHECK (protein >= 0),
    carbs NUMERIC(8, 2) NOT NULL CHECK (carbs >= 0),
    fat NUMERIC(8, 2) NOT NULL CHECK (fat >= 0),
    confidence NUMERIC(3, 2) NOT NULL CHECK (confidence BETWEEN 0 AND 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_food_items_food_log_id ON food_items (food_log_id);