/*
SHT3x - Connecting to the SHT3x sensor.
Copyright (C) 2022, The Cacophony Project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"errors"
	"log"
	"time"

	arg "github.com/alexflint/go-arg"
	"github.com/snksoft/crc"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

const (
	i2cAddress      = 0x44
	maxTxAttempts   = 3
	txRetryInterval = time.Second
)

var version = "No version provided"

type argSpec struct {
	I2cAddress uint16 `arg:"--address" help:"Address of MMC5603NJ sensor"`
}

func (argSpec) Version() string {
	return version
}

func procArgs() argSpec {
	// Set argument default values.
	args := argSpec{
		I2cAddress: i2cAddress,
	}
	arg.MustParse(&args)
	return args
}

func main() {
	err := runMain()
	if err != nil {
		log.Fatal(err.Error())
	}
}

func runMain() error {
	args := procArgs()
	log.SetFlags(0) // Removes default timestamp flag
	log.Printf("running version: %s", version)

	if _, err := host.Init(); err != nil {
		return err
	}
	log.Println("connecting to i2c bus")
	bus, err := i2creg.Open("")
	if err != nil {
		return err
	}

	log.Printf("connecting to Humidity sensor on address '%x'", args.I2cAddress)
	dev := &i2c.Dev{Bus: bus, Addr: args.I2cAddress}
	if dev == nil {
		bus.Close()
		return errors.New("failed to connect to device")
	}
	log.Println("connected")
	s := SHT3x{
		dev: dev,
	}
	s.MakeReading()
	return nil
}

type SHT3x struct {
	dev *i2c.Dev
}

func (s SHT3x) MakeReading() error {
	//Send make reading command
	if err := s.tx([]byte{0x24, 0x00}, nil); err != nil {
		return err
	}
	time.Sleep(30 * time.Millisecond)
	data := make([]byte, 6)
	if err := s.tx(nil, data); err != nil {
		return err
	}
	log.Println(data)

	crcTable := crc.NewTable(&crc.Parameters{
		Width:      8,
		Polynomial: 0x31,
		ReflectIn:  false,
		ReflectOut: false,
		Init:       0xFF,
		FinalXor:   0x00,
	})

	// Check CRC
	if crcTable.CalculateCRC(data[0:2]) != uint64(data[2]) {
		return errors.New("crc for temp does not match")
	}
	if crcTable.CalculateCRC(data[3:5]) != uint64(data[5]) {
		return errors.New("crc for humidity does not match")
	}
	var tempRaw, humidityRaw uint16
	tempRaw |= uint16(data[0]) << 8
	tempRaw |= uint16(data[1])
	log.Println(tempRaw)

	humidityRaw |= uint16(data[3]) << 8
	humidityRaw |= uint16(data[4])
	log.Println(humidityRaw)

	temp := float32(-45 + 175*float32(tempRaw)/float32(65535))
	log.Println(temp)

	humidity := float32(-45 + 175*float32(humidityRaw)/float32(65535))
	log.Println(humidity)

	return nil
}

func (m SHT3x) tx(write, read []byte) error {
	attempts := 0
	for {
		err := m.dev.Tx(write, read)
		if err == nil {
			return nil
		}
		attempts++
		if attempts >= maxTxAttempts {
			return err
		}
		time.Sleep(txRetryInterval)
	}
}
