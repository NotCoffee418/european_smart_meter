-- +up
-- Live power aggregates (wh) averages by hour, day, month
-- Consumption of a given hour/day/month
CREATE TABLE aggregate_live_power_hourly (
    -- Unix timestamp of start of hour
    hour_start INTEGER PRIMARY KEY,
    consumption_day_wh INTEGER NOT NULL DEFAULT 0,
    consumption_night_wh INTEGER NOT NULL DEFAULT 0,
    production_day_wh INTEGER NOT NULL DEFAULT 0,
    production_night_wh INTEGER NOT NULL DEFAULT 0,
    sample_count INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE aggregate_live_power_daily (
    -- Unix timestamp of start of day
    day_start INTEGER PRIMARY KEY,
    consumption_day_wh INTEGER NOT NULL DEFAULT 0,
    consumption_night_wh INTEGER NOT NULL DEFAULT 0,
    production_day_wh INTEGER NOT NULL DEFAULT 0,
    production_night_wh INTEGER NOT NULL DEFAULT 0,
    sample_count INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE aggregate_live_power_monthly (
    -- Unix timestamp of start of month
    month_start INTEGER PRIMARY KEY,
    consumption_day_wh INTEGER NOT NULL DEFAULT 0,
    consumption_night_wh INTEGER NOT NULL DEFAULT 0,
    production_day_wh INTEGER NOT NULL DEFAULT 0,
    production_night_wh INTEGER NOT NULL DEFAULT 0,
    sample_count INTEGER NOT NULL DEFAULT 0
);

-- Meter standings in wh of power at any given hour
-- May need to be inferred from nearest total_power_readings
CREATE TABLE snapshot_total_power_hourly(
    -- Must be rounded to start of hour
    timestamp INTEGER PRIMARY KEY,
    consumption_day_standing INTEGER NOT NULL DEFAULT 0,
    consumption_night_standing INTEGER NOT NULL DEFAULT 0,
    production_day_standing INTEGER NOT NULL DEFAULT 0,
    production_night_standing INTEGER NOT NULL DEFAULT 0
);
-- Snapshot total gas entries to retain
-- May need to be inferred from nearest total_gas_readings
CREATE TABLE snapshot_total_gas_hourly (
    -- Must be rounded to start of hour
    timestamp INTEGER PRIMARY KEY,
    dm3_standing INTEGER NOT NULL DEFAULT 0
);

-- +down
DROP TABLE aggregate_live_power_hourly;
DROP TABLE aggregate_live_power_daily;
DROP TABLE aggregate_live_power_monthly;
DROP TABLE snapshot_total_power_hourly;
DROP TABLE snapshot_total_gas_hourly;
