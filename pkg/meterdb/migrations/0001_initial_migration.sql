-- +up
CREATE TABLE power_readings (
    timestamp INTEGER PRIMARY KEY,
    watt INTEGER NOT NULL,
    reading_type INTEGER NOT NULL,
    total_consumption_watt_hours INTEGER NOT NULL,
    total_production_watt_hours INTEGER NOT NULL
);

CREATE TABLE gas_readings (
    timestamp INTEGER PRIMARY KEY,
    cubic_meter INTEGER NOT NULL
);

-- +down
DROP TABLE power_readings;
DROP TABLE gas_readings;

