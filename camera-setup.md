# Thermal camera 2 setup process

## Flash Image
- Download latest 64 bit image from https://www.raspberrypi.com/software/operating-systems/
- Using the RPi Imager https://www.raspberrypi.com/software/ 

## Prepare image
- Make `/etc/cacophony/config.toml`
- Install salt


## RTC
- Add `dtoverlay=i2c-rtc,pcf8563` at the end of `/boot/config.txt`
- Reboot and run `i2cdetect -y 1` you should see `UU` at `0x51`. This shows that the RTC loaded properly.



## Golang Software to install
For each of these check out at the given branch and use goreleaser to build a .deb file and copy that to the device to install.

### device-register
- tc2 branch //TODO

### thermal-recorder
- tc2 branch //TODO

### thermal-uploader
- tc2 branch //TODO

### event-reporter
- tc2 branch //TODO

### audiobait 
- tc2 branch //TODO

### modemd
- tc2 branch //TODO

### humidity sensor
- //TODO

### audio-recorder
- //TODO

### magnet sensor
- //TODO

### salt updater
- //TODO

### cacophony-config
- //TODO

### management-interface 
- //TODO
