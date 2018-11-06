package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/stianeikeland/go-rpio"
	//"github.com/drahcirennobran/go-rpio-mock"
)

const (
	wheelSize_mm = 80
	whellDist    = 200
	stepPerTurn  = 200
	ticksPerStep = 8
	vmax         = 50

	LEFT  int = 1
	RIGHT int = 2
	CW    int = 3
	CCW   int = 4
	FW    int = 5
	BW    int = 6
	TL    int = 7
	TR    int = 8

	pinStepRight      = rpio.Pin(16) //36
	pinDirectionRight = rpio.Pin(20) //38
	pinDisableRight   = rpio.Pin(21) //40
	pinStepLeft       = rpio.Pin(13) //33
	pinDirectionLeft  = rpio.Pin(19) //35
	pinDisableLeft    = rpio.Pin(26) //37

	pinVoid = rpio.Pin(31)
)

type Command struct {
	Instruction int
	Iteration   int
	Pause       float64
}

type SpeedCommand struct {
	Instruction int
	Speed_mmps  int
	dist_mm     int
	angle       int
}

func initGpio() {
	pinStepRight.Output()
	pinDirectionRight.Output()
	pinDisableRight.Output()
	pinStepLeft.Output()
	pinDirectionLeft.Output()
	pinDisableLeft.Output()
}
func enableWheels() {
	pinDisableRight.Low()
	pinDisableLeft.Low()
}
func disableWheels() {
	pinDisableRight.High()
	pinDisableLeft.High()
}
func processSmoothSpeedCommand(smoothCmdChan chan SpeedCommand, cmdChan chan SpeedCommand) {
	for {
		command := <-smoothCmdChan
		v := command.Speed_mmps
		if command.Speed_mmps > vmax {
			v = vmax
		}
		speedIncrement := 2
		stepDist := 10
		accelDist := command.dist_mm / 2
		fmt.Printf("accelDist %v\n", accelDist)
		stepNb := 0
		switch command.Instruction {
		case FW, BW:
			fmt.Printf("smoothCommand : %v speed %v mm/s along %v mm\n", command.Instruction, command.Speed_mmps, command.dist_mm)
			dist := stepDist
			speed := speedIncrement
			for ; dist < accelDist && speed < v; dist, speed = dist+stepDist, speed+speedIncrement {
				//fmt.Printf("speed %v ; stepDist %v ; (dist %v)\n", speed, stepDist, dist)
				cmdChan <- SpeedCommand{command.Instruction, speed, stepDist, 0}
				stepNb++
			}
			stableDist := command.dist_mm - 2*(dist-stepDist)
			if stableDist > 0 {
				//fmt.Printf("speed %v ; dist %v ; \n", speed, stableDist)
				cmdChan <- SpeedCommand{command.Instruction, speed, stableDist, 0}
			}
			for i := 0; i < stepNb; i++ {
				speed -= speedIncrement
				//fmt.Printf("speed %v : stepDist %v ; \n", speed, stepDist)
				cmdChan <- SpeedCommand{command.Instruction, speed, stepDist, 0}
			}
		/*
			case TL:
						fmt.Printf("smoothCommand : TL %d\n", command.Iteration)
					case TR:
						fmt.Printf("smoothCommand : TR %d\n", command.Iteration)
		*/
		default:
			fmt.Printf("processSmoothCommand unknown command %d\n", command.Instruction)
		}
	}
}
func processSpeedCommand(cmdChan chan SpeedCommand, leftChan chan SpeedCommand, rightChan chan SpeedCommand) {
	for {
		command := <-cmdChan
		switch command.Instruction {
		case FW:
			fmt.Printf("processSpeedCommand FW speed %v mm/s along %v mm\n", command.Speed_mmps, command.dist_mm)
			leftChan <- SpeedCommand{CW, command.Speed_mmps, command.dist_mm, 0}
			rightChan <- SpeedCommand{CCW, command.Speed_mmps, command.dist_mm, 0}
		case BW:
			fmt.Printf("processSpeedCommand BW speed %v mm/s along %v mm\n", command.Speed_mmps, command.dist_mm)
			leftChan <- SpeedCommand{CCW, command.Speed_mmps, command.dist_mm, 0}
			rightChan <- SpeedCommand{CW, command.Speed_mmps, command.dist_mm, 0}
		case TL:
			fmt.Printf("processSpeedCommand TL speed %v mm/s along %v degrees\n", command.Speed_mmps, command.angle)
			dist := (math.Pi * whellDist * float64(command.angle)) / 360
			leftChan <- SpeedCommand{CCW, command.Speed_mmps, int(dist), 0}
			rightChan <- SpeedCommand{CCW, command.Speed_mmps, int(dist), 0}
		case TR:
			dist := (math.Pi * whellDist * float64(command.angle)) / 360
			fmt.Printf("processSpeedCommand TL speed %v mm/s along %v degrees\n", command.Speed_mmps, command.angle)
			leftChan <- SpeedCommand{CW, command.Speed_mmps, int(dist), 0}
			rightChan <- SpeedCommand{CW, command.Speed_mmps, int(dist), 0}
		default:
			fmt.Printf("processCommand unknown command %d\n", command.Instruction)
		}
	}
}

func processWheel(side int, c chan SpeedCommand) {
	for {
		command := <-c

		pinStep := pinVoid
		pinDirection := pinVoid
		if side == LEFT {
			pinStep = pinStepLeft
			pinDirection = pinDirectionLeft
		} else if side == RIGHT {
			pinStep = pinStepRight
			pinDirection = pinDirectionRight
		} else {
			fmt.Println("ni droite ni gauche, bien au contraire")
		}

		switch command.Instruction {
		case CW:
			//fmt.Printf("CW %d %f\n", command.Iteration, command.Pause)
			pinDirection.High()
		case CCW:
			//fmt.Printf("CCW %d %f\n", command.Iteration, command.Pause)
			pinDirection.Low()
		default:
			fmt.Printf("processWheel2 unknown command %d\n", command.Instruction)
		}
		steppersMove(pinStep, command.Speed_mmps, command.dist_mm)
	}
}

func steppersMove(pin rpio.Pin, speed_mmps int, dist_mm int) {
	iterations := float64(stepPerTurn*ticksPerStep*dist_mm) / (math.Pi * wheelSize_mm)
	pause := 500 * float64(dist_mm) / float64(speed_mmps)
	//fmt.Printf("speed %v, dist %d\n", speed_mmps, dist_mm)

	steppersTicks(pin, int(iterations), int(pause))
}

func steppersTicks(pin rpio.Pin, iterations int, pause int) {
	//fmt.Printf("iterations %v, pause %v\n", iterations, pause)
	for i := 0; i < iterations; i++ {
		//fmt.Printf(".")
		pin.High()
		time.Sleep(time.Microsecond * time.Duration(pause))
		pin.Low()
		time.Sleep(time.Microsecond * time.Duration(pause))
	}
}
func main() {
	if err := rpio.Open(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer rpio.Close()

	var input string
	initGpio()
	enableWheels()

	smoothSpeedCommandChan := make(chan SpeedCommand)
	speedCommandChan := make(chan SpeedCommand)
	leftWhellSpeedChan := make(chan SpeedCommand)
	rightWhellSpeedChan := make(chan SpeedCommand)

	go processSmoothSpeedCommand(smoothSpeedCommandChan, speedCommandChan)
	go processSpeedCommand(speedCommandChan, leftWhellSpeedChan, rightWhellSpeedChan)
	go processWheel(LEFT, leftWhellSpeedChan)
	go processWheel(RIGHT, rightWhellSpeedChan)

	//leftWhellSpeedChan <- SpeedCommand{CW, 10, 60}
	//speedCommandChan <- SpeedCommand{FW, 10, 100}

	//smoothSpeedCommandChan <- SpeedCommand{FW, 100, 500, 0}
	smoothSpeedCommandChan <- SpeedCommand{TL, 100, 0, 45}
	//smoothSpeedCommandChan <- SpeedCommand{BW, 100, 500, 0}
	//speedCommandChan <- SpeedCommand{TR, 40, 0, 45}

	fmt.Scanln(&input)

	disableWheels()
}
