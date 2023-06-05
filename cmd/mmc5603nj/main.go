/*
MMC5603NJ - Connect to MMC5603NJ sensor.
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
	"math"
	"time"

	arg "github.com/alexflint/go-arg"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

const (
	i2cAddress      = 0x30
	maxTxAttempts   = 3
	txRetryInterval = time.Second

	MMC5603NJ_ADDR_INTCTRL0  = 0x1B
	MMC5603NJ_ADDR_INTCTRL1  = 0x1C
	MMC5603NJ_ADDR_INTCTRL2  = 0x1D
	MMC5603NJ_ADDR_PRODUCTID = 0x39
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

	log.Printf("connecting to MMC5603NJ on address '%x'", args.I2cAddress)
	dev := &i2c.Dev{Bus: bus, Addr: args.I2cAddress}
	if dev == nil {
		bus.Close()
		return errors.New("failed to connect to device")
	}
	log.Println("connected")
	m := MMC5603NJ{
		dev: dev,
	}

	log.Print("running a software reset.. ")
	if err := m.SoftwareReset(); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	log.Println("done")

	id, err := m.readProductId()
	if err != nil {
		return err
	}
	log.Printf("product ID: %d", int(id))

	log.Print("disable continious mode.. ")
	if err := m.DisableContinuousMode(); err != nil {
		return err
	}
	log.Println("done")

	m.ReadGauss()

	return nil
}

type MMC5603NJ struct {
	dev *i2c.Dev
}

func (m MMC5603NJ) readProductId() (byte, error) {
	id := make([]byte, 1)
	err := m.tx([]byte{MMC5603NJ_ADDR_PRODUCTID}, id)
	return id[0], err
}

func (m MMC5603NJ) SoftwareReset() error {
	_, err := m.dev.Write([]byte{MMC5603NJ_ADDR_INTCTRL1, byte(0b10000000)})
	return err
}

func (m MMC5603NJ) DisableContinuousMode() error {
	_, err := m.dev.Write([]byte{MMC5603NJ_ADDR_INTCTRL2, 0})
	return err
}

func (m MMC5603NJ) ReadGauss() (x, y, z, mag float32, err error) {
	// Request new data
	m.dev.Write([]byte{0x1b, byte(0b00000001)})
	// Wait for reading
	time.Sleep(10 * time.Millisecond)

	data := make([]byte, 9)
	m.tx([]byte{0x00}, data)
	var xRaw uint32
	xRaw |= uint32(data[0]) << 12
	xRaw |= uint32(data[1]) << 4
	xRaw |= uint32(data[6]) << 0
	x = float32(int32(xRaw)-524288) * 0.0625

	var yRaw uint32
	yRaw |= uint32(data[2]) << 12
	yRaw |= uint32(data[3]) << 4
	yRaw |= uint32(data[7]) << 0
	y = float32(int32(yRaw)-524288) * 0.0625

	var zRaw uint32
	zRaw |= uint32(data[4]) << 12
	zRaw |= uint32(data[5]) << 4
	zRaw |= uint32(data[8]) << 0
	z = float32(int32(zRaw)-524288) * 0.0625

	log.Printf("X: %f", x)
	log.Printf("Y: %f", y)
	log.Printf("Z: %f", z)
	log.Println(math.Atan(float64(x/z)) * 180 / math.Pi)
	return
}

func (m MMC5603NJ) tx(write, read []byte) error {
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
