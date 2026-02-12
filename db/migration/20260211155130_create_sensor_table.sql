-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS sensor_readings (
    time            TIMESTAMPTZ NOT NULL,
    sensor_id       TEXT NOT NULL,
    zone            TEXT NOT NULL,
    temperature     DOUBLE PRECISION,
    humidity        DOUBLE PRECISION,
    co_level        DOUBLE PRECISION,
    battery_level   DOUBLE PRECISION
);

CREATE INDEX ON sensor_readings (sensor_id, time DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sensor_readings;
-- +goose StatementEnd