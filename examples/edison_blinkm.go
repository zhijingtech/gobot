//go:build example
// +build example

//
// Do not build by default.

package main

import (
	"fmt"
	"time"

	"gobot.io/x/gobot/v2"
	"gobot.io/x/gobot/v2/drivers/i2c"
	"gobot.io/x/gobot/v2/platforms/intel-iot/edison"
)

func main() {
	e := edison.NewAdaptor()
	blinkm := i2c.NewBlinkMDriver(e)

	work := func() {
		gobot.Every(3*time.Second, func() {
			r := byte(gobot.Rand(255))
			g := byte(gobot.Rand(255))
			b := byte(gobot.Rand(255))
			if err := blinkm.Rgb(r, g, b); err != nil {
				fmt.Println(err)
			}
			color, err := blinkm.Color()
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println("color", color)
		})
	}

	robot := gobot.NewRobot("blinkmBot",
		[]gobot.Connection{e},
		[]gobot.Device{blinkm},
		work,
	)

	if err := robot.Start(); err != nil {
		panic(err)
	}
}
