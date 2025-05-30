-- +up
CREATE TABLE live_power_readings (
    timestamp INTEGER PRIMARY KEY,
    watt INTEGER NOT NULL,
    reading_type INTEGER NOT NULL
);

CREATE TABLE total_power_readings (
    timestamp INTEGER PRIMARY KEY,
    watthour INTEGER NOT NULL,
    reading_type INTEGER NOT NULL
);

CREATE TABLE total_gas_readings (
    timestamp INTEGER PRIMARY KEY,
    consumption_dm3 INTEGER NOT NULL
);

-- +down
DROP TABLE live_power_readings;
DROP TABLE total_power_readings;
DROP TABLE total_gas_readings;

