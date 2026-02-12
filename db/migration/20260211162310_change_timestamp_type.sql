-- +goose Up
-- +goose StatementBegin
ALTER TABLE sensor_readings 
    ALTER COLUMN time TYPE BIGINT 
    USING EXTRACT(EPOCH FROM time)::BIGINT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sensor_readings 
    ALTER COLUMN time TYPE TIMESTAMPTZ 
    USING TO_TIMESTAMP(time);
-- +goose StatementEnd