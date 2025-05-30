# Smart Meter API

This is a simple API to get data from the European smart meter.  
This was designed for Belgian meters but should work for other countries as well.

## Requirements

- Raspberry Pi 3+ (or other Linux machine)
- [P1 Smart Meter Cable](https://webshop.cedel.nl/nl/Slimme-meter-kabel-P1-naar-USB)
- [Activated P1 port on your smart meter](https://www.stroohm.be/en/help/using-and-accessing-the-p1-port-of-the-digital-meter-in-belgium/)

## Installation

1. [Set up Raspberry Pi](https://www.raspberrypi.com/documentation/computers/getting-started.html)
2. Connect the cable to the P1 port on the meter and a USB port on the Raspberry Pi
3. SSH into the Raspberry Pi or open a terminal on the Raspberry Pi
4. Disable Pi Desktop (optional, but recommended on older Pi's)  
    When running additional smart meter related software on the Pi, it may cause memory issues if we don't.

    ```bash
    sudo raspi-config
    # System Options → Boot / Auto Login → Console
    # Exit and reboot
    ```

5. Install Interpreter API

    ```bash
    curl -sSL https://raw.githubusercontent.com/NotCoffee418/european_smart_meter/main/installers/install_interpreter.sh | sudo bash /dev/stdin
    ```

### Docker Compose (Recommended)

```bash
git clone https://github.com/NotCoffee418/belgian_smart_meter_api
cd belgian_smart_meter_api
docker compose up -d
```

### Directly (Not recommended)

```bash
git clone https://github.com/NotCoffee418/belgian_smart_meter_api
cd belgian_smart_meter_api
python3 -m venv .venv
source .venv/bin/activate
pip3 install -r requirements.txt
python3 main.py
```

## Test the connection

```bash
# Run this command from the Raspberry Pi.
# You should see a JSON response with the latest data from the smart meter.
curl http://localhost:9039/latest | python3 -m json.tool
```



## API Usage

Additional tools for the API (still being built or available in my github if i forget to update this message)


## Endpoints

- **/latest**: Get the latest data from the smart meter
- **/ws**: Subscribe to the websocket endpoint to get real-time data from the smart meter

Both output the following JSON response structure:

```json
{
  "timestamp": "2025-05-30T15:52:07", // Local Time
  "current_consumption_kw": 0.150, // Combined Consumption (L1+L2+L3)
  "current_production_kw": 0.0,
  "l1_consumption_kw": 0.150,
  "l2_consumption_kw": 0.0, // L2 and L3 is for 3 phase meters (industrial)
  "l3_consumption_kw": 0.0, // L2 and L3 is for 3 phase meters (industrial)
  "l1_production_kw": 0.0,
  "l2_production_kw": 0.0,
  "l3_production_kw": 0.0,
  "total_consumption_day_kwh": 9999.999,
  "total_consumption_night_kwh": 9999.99,
  "total_production_day_kwh": 9999.99,
  "total_production_night_kwh": 9999.99,
  "current_tariff": 1, // 1=day, 2=night
  "l1_voltage_v": 237.8,
  "l2_voltage_v": 0.0,
  "l3_voltage_v": 0.0,
  "l1_current_a": 2.28,
  "l2_current_a": 0.0,
  "l3_current_a": 0.0,
  "switch_electricity": 1, // 1=on, 0=off - Physical switch on the meter
  "switch_gas": 1, // 1=on, 0=off - Physical switch on the meter
  "meter_serial_electricity": "XXXXXXXXX",
  "meter_serial_gas": "XXXXXXXXX",
  "gas_consumption_m3": 9999.99 // Updated every 10 minutes
}
```


## Uninstallation

```bash
# Interpreter API (everything else depends on this)
curl -sSL https://raw.githubusercontent.com/NotCoffee418/european_smart_meter/main/installers/uninstall_interpreter.sh | sudo bash /dev/stdin
```

## Troubleshooting

### Device not found

If you have multiple USB ports, you need to check which one is the P1 port and adjust it in the docker-compose.yml file if it's not /dev/ttyUSB0.

```bash
# List all USB devices
ls /dev/ttyUSB*
```

### No data received
It may be that you have submitted the request to activate the P1 port but it's not activated yet.
You can check if you're receiving data by running the following command:

```bash
python3 -m serial.tools.miniterm /dev/ttyUSB0 115200 --xonxoff
```

