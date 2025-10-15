-- +up
-- Live power aggregates (average watt) by hour, day, month
-- Average power consumption of a given hour/day/month
CREATE TABLE aggregate_live_power_hourly (
    -- Unix timestamp of start of hour
    start_time INTEGER PRIMARY KEY,
    consumption_day_wh INTEGER NOT NULL DEFAULT 0,
    consumption_day_sample_count INTEGER NOT NULL DEFAULT 0,
    consumption_night_wh INTEGER NOT NULL DEFAULT 0,
    consumption_night_sample_count INTEGER NOT NULL DEFAULT 0,
    production_day_wh INTEGER NOT NULL DEFAULT 0,
    production_day_sample_count INTEGER NOT NULL DEFAULT 0,
    production_night_wh INTEGER NOT NULL DEFAULT 0,
    production_night_sample_count INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE aggregate_live_power_daily (
    -- Unix timestamp of start of day
    start_time INTEGER PRIMARY KEY,
    consumption_day_wh INTEGER NOT NULL DEFAULT 0,
    consumption_day_sample_count INTEGER NOT NULL DEFAULT 0,
    consumption_night_wh INTEGER NOT NULL DEFAULT 0,
    consumption_night_sample_count INTEGER NOT NULL DEFAULT 0,
    production_day_wh INTEGER NOT NULL DEFAULT 0,
    production_day_sample_count INTEGER NOT NULL DEFAULT 0,
    production_night_wh INTEGER NOT NULL DEFAULT 0,
    production_night_sample_count INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE aggregate_live_power_monthly (
    -- Unix timestamp of start of month
    start_time INTEGER PRIMARY KEY,
    consumption_day_wh INTEGER NOT NULL DEFAULT 0,
    consumption_day_sample_count INTEGER NOT NULL DEFAULT 0,
    consumption_night_wh INTEGER NOT NULL DEFAULT 0,
    consumption_night_sample_count INTEGER NOT NULL DEFAULT 0,
    production_day_wh INTEGER NOT NULL DEFAULT 0,
    production_day_sample_count INTEGER NOT NULL DEFAULT 0,
    production_night_wh INTEGER NOT NULL DEFAULT 0,
    production_night_sample_count INTEGER NOT NULL DEFAULT 0
);
-- Standing and increase vs previous snapshot
CREATE TABLE snapshot_total_power_hourly(
    -- Must be rounded to start of hour
    timestamp INTEGER PRIMARY KEY,
    consumption_day_standing INTEGER NOT NULL DEFAULT 0,
    consumption_day_increase INTEGER NOT NULL DEFAULT 0,
    consumption_night_standing INTEGER NOT NULL DEFAULT 0,
    consumption_night_increase INTEGER NOT NULL DEFAULT 0,
    production_day_standing INTEGER NOT NULL DEFAULT 0,
    production_day_increase INTEGER NOT NULL DEFAULT 0,
    production_night_standing INTEGER NOT NULL DEFAULT 0,
    production_night_increase INTEGER NOT NULL DEFAULT 0
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