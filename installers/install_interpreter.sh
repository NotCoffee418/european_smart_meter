if [ "$EUID" -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi

# Permissions for serial port
sudo usermod -a -G dialout $USER
sudo usermod -a -G tty $USER


