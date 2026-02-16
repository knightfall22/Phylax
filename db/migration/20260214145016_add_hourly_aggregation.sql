-- +goose Up
CREATE TABLE IF NOT EXISTS sensor_hourly_stats (
    bucket       TIMESTAMPTZ NOT NULL,
    zone         TEXT NOT NULL,
    avg_temp     DOUBLE PRECISION,
    max_temp     DOUBLE PRECISION,
    min_temp     DOUBLE PRECISION,
    avg_humidity DOUBLE PRECISION,
    sample_count INT,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (bucket, zone)
);

CREATE INDEX IF NOT EXISTS idx_hourly_stats_time ON sensor_hourly_stats(bucket);

-- +goose StatementBegin
CREATE OR REPLACE PROCEDURE run_hourly_rollup()
LANGUAGE SQL
AS $$
    INSERT INTO sensor_hourly_stats (bucket, zone, avg_temp, max_temp, min_temp, avg_humidity, sample_count)
    SELECT 
        date_trunc('hour', to_timestamp(time / 1000.0)) as bucket, 
        zone,
        AVG(temperature),
        MAX(temperature),
        MIN(temperature),
        AVG(humidity),
        COUNT(*)
    FROM sensor_readings
    WHERE to_timestamp(time / 1000.0) < date_trunc('hour', NOW())
    GROUP BY 1, 2
    ON CONFLICT (bucket, zone) DO NOTHING;

    DELETE FROM sensor_readings 
    WHERE time < (EXTRACT(EPOCH FROM (NOW() - INTERVAL '24 hours')) * 1000);
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP PROCEDURE IF EXISTS run_hourly_rollup;
-- +goose StatementEnd
DROP TABLE IF EXISTS sensor_hourly_stats;