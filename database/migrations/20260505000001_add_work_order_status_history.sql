-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS "work_order_status_history" (
  "id" uuid PRIMARY KEY,
  "work_order_id" uuid NOT NULL,
  "from_status" varchar(30) NOT NULL,
  "to_status" varchar(30) NOT NULL,
  "changed_by_user_id" uuid,
  "note" text,
  "changed_at" timestamp NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_work_order_status_history_wo_id ON "work_order_status_history" ("work_order_id");
CREATE INDEX IF NOT EXISTS idx_work_order_status_history_changed_at ON "work_order_status_history" ("changed_at");

DO $$ BEGIN
  ALTER TABLE "work_order_status_history" ADD CONSTRAINT fk_wo_status_history_work_order
    FOREIGN KEY ("work_order_id") REFERENCES "work_orders" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
  ALTER TABLE "work_order_status_history" ADD CONSTRAINT fk_wo_status_history_changed_by
    FOREIGN KEY ("changed_by_user_id") REFERENCES "users" ("id") DEFERRABLE INITIALLY IMMEDIATE;
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "work_order_status_history";
-- +goose StatementEnd
