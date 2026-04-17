CREATE TABLE IF NOT EXISTS supplies (
    id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    service_id UUID    NOT NULL,
    item_id    UUID    NOT NULL,
    quantity   INTEGER NOT NULL
);
