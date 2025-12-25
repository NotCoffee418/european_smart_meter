# Smart Meter API

Modular services to interact with the European smart meter designed to run on a Raspberry Pi.

- Interpreter API: Read the meter and provides the values as an API.
- Meter Collector: Get data from Interpreter API and store it in the database.

This was designed for Belgian meters but should work for other countries as well.

## Current state
This tool has expanded to be rather integrated with my own personal setup, it will need to be customized for your own setup.  
It's data collecting features work, but other parts may be half implemented or missing features.  
I currently only use this as a gateway between my smart meter and my [power_control_center](https://github.com/NotCoffee418/power_control_center) tool which manages airco units based on live readings from this API.  
Additionally, there is the needless baggage of getting data off my specific solar inverter. Use the code freely, but I would strip it or only use specific modules for your own purposes.

## Requirements

- Raspberry Pi 3+ (or other Linux machine)
- [P1 Smart Meter Cable](https://webshop.cedel.nl/nl/Slimme-meter-kabel-P1-naar-USB)
- [Activated P1 port on your smart meter](https://www.stroohm.be/en/help/using-and-accessing-the-p1-port-of-the-digital-meter-in-belgium/)

## Set up networking
- Connect the ethernet cable to the router
- Adjust wifi settings to connect to solar panel.

### Configure wireless credentials
```bash
sudo nano /etc/NetworkManager/system-connections/preconfigured.nmconnection
```
Adjust the credentials in the file. Ensure no properties are missing or the connection will just drop out and not recover after a few days.
```ini
[connection]
id=preconfigured
uuid=58af4df0-76cf-4270-957d-1f8c5773051d
type=wifi
autoconnect-retrues=0
interface-name=wlan0 #change if needed
timestamp=1758825933

[wifi]
mode=infrastructure
ssid=XXXXXXXXXXXXXXXXXXXXX

[wifi-security]
key-mgmt=wpa-psk
psk=XXXXXXXXXXXXXXXXXXXXXX
auth-timeout=30

[ipv4]
method=auto

[ipv6]
method=disabled

[proxy]
```
```bash
sudo nmcli connection reload
sudo nmcli connection down preconfigured
sudo nmcli connection up preconfigured
# Check wifi connection
iwgetid -r # Should be SUNXXXX
```

### Set up firewall for wlan
Block everything on wlan to avoid solar being used as an access point to the network.  
Check device names to confirm with `nmcli connection show`
```bash
# Install ufw
sudo apt-get install ufw

# Disable first to configure safely
sudo ufw disable

# Default policies - deny everything
sudo ufw default deny incoming
sudo ufw default deny outgoing

# Allow everything on eth0 (or whatever your main interface is)
sudo ufw allow in on eth0
sudo ufw allow out on eth0

# Allow loopback
sudo ufw allow in on lo
sudo ufw allow out on lo

# Block ALL incoming on wlan0
sudo ufw deny in on wlan0

# Allow ICMP out on wlan0 to specific IP
sudo ufw allow out on wlan0 to 192.168.200.1

# And ping too
sudo ufw allow out on wlan0 proto icmp to 192.168.200.1

# Enable ufw
sudo ufw --force enable

# Check status
sudo ufw status verbose
```

```bash
# Check wifi connection
iwgetid -r

# Confirm solar inverter IP is reachable
ping 192.168.200.1
```

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
    # Install or update the Interpreter API
    sudo curl -sSL https://raw.githubusercontent.com/NotCoffee418/european_smart_meter/main/installers/install_interpreter.sh | sudo bash /dev/stdin
    ```

6. Install Meter Collector

    ```bash
    # Install or update the Meter Collector
    sudo curl -sSL https://raw.githubusercontent.com/NotCoffee418/european_smart_meter/main/installers/install_meter_collector.sh | sudo bash /dev/stdin
    ```

7. Set up Auto-Upgrading and install Grafana

    ```bash
    # Install or update the Auto-Updater
    sudo curl -sSL https://raw.githubusercontent.com/NotCoffee418/european_smart_meter/refs/heads/main/installers/autoupdater.sh | sudo bash /dev/stdin
    ```

## Updating

Run the install scripts again to update the services. Configs remain intact.

## Interpreter API
The Interpreter API reads the meter and provides the values as an API with a websocket endpoint option.  
This service is essential for all other services to work and can be optionally run as a standalone application to work with custom software.

### Endpoints

- **/latest**: Get the latest data from the smart meter
- **/ws**: Subscribe to the websocket endpoint to get real-time data from the smart meter
- **/solar**: Get current power production from solar inverter

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

# Meter Collector
curl -sSL https://raw.githubusercontent.com/NotCoffee418/european_smart_meter/main/installers/uninstall_meter_collector.sh | sudo bash /dev/stdin
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

### Solar inverter wifi not connecting
Only accepts 1 device.  
```bash
# On Pi if debugging on pc
sudo nmcli connection down preconfigured
```

## Unsorted info (todo)
### Paths
- /etc/european_smart_meter/interpreter_api.toml
- /etc/european_smart_meter/meter_collector.toml
- /var/lib/european_smart_meter/esm-meter.db
- /usr/bin/european_smart_meter/interpreter_api
- /usr/bin/european_smart_meter/meter_collector
