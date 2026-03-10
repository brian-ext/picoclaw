# Board Pinout References

This file documents I2C/SPI pinmux setup commands for supported boards.

## Notes

- Some boards share I2C/SPI pins with WiFi SDIO.
- You must configure pinmux before running I2C/SPI operations.

## LicheeRV Nano (example)

1. Stop WiFi: `/etc/init.d/S30wifi stop`
2. Load module: `modprobe i2c-dev`
3. Configure pinmux using `devmem` (board-specific)
4. Verify: `i2c detect` and `i2c scan`
