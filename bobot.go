package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/stianeikeland/go-rpio"
	//"github.com/drahcirennobran/go-rpio-mock"
)

var (
	pause [][]float64 = [][]float64{{1000, 2}, {500, 2}, {250, 4}, {100, 10}, {50, 20}, {33.3, 33}, {25, 40}, {20, 50}, {16.67, 60}, {14.29, 70}, {12.5, 80}, {11.11, 90}, {10, 100}}
)

const (
	wheelSize_mm = 80
	stepPerTurn  = 200
	ticksPerStep = 16
	wheelSpacing = 0.2

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
	Speed_mmps  int32
	Duration_ms int32
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
func SplitAcceleration(command Command) []Command {
	splittedCommands := make([]Command, 0)
	for i, totalTicks := 0, 0; pause[i][0] >= command.Pause && i < len(pause) && totalTicks < command.Iteration; i, totalTicks = i+1, totalTicks+int(pause[i][1]) {
		var ticks int
		if totalTicks+int(pause[i][1]) < command.Iteration && pause[i][0] > command.Pause {
			ticks = int(pause[i][1])
		} else {
			ticks = command.Iteration - totalTicks
		}
		splittedCommands = append(splittedCommands, Command{0, ticks, pause[i][0]})
		//fmt.Printf("i=%d ; %d ticks, pause de %f (totalticks=%d)\n", i, ticks, pause[i][0], totalTicks)
	}
	return splittedCommands
}
func processSmoothSpeedCommand(smoothCmdChan chan SpeedCommand, cmdChan chan SpeedCommand) {
	for {
		command := <-smoothCmdChan
		switch command.Instruction {
		case FW:
			fmt.Printf("smoothCommand : FW speed %v mm/s during %v ms\n", command.Speed_mmps, command.Duration_ms)
			/*
				for speed := 1; speed < command.Speed_mmps; speed++ {
					cmdChan <- SpeedCommand{FW, speed, 100}
				}
			*/
			/*
				accelerationCommands := SplitAcceleration(command)
				for _, splittedCommand := range accelerationCommands {
					cmdChan <- Command{FW, splittedCommand.Iteration, splittedCommand.Pause}
				}
			*/
		case BW:
			fmt.Printf("smoothCommand : BW speed %v mm/s during %v ms\n", command.Speed_mmps, command.Duration_ms)
		//TODO RELATIVE_FW from last speed to target speed
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
			fmt.Printf("FW speed %v mm/s during %v ms\n", command.Speed_mmps, command.Duration_ms)
			leftChan <- SpeedCommand{CW, command.Speed_mmps, command.Duration_ms}
			rightChan <- SpeedCommand{CCW, command.Speed_mmps, command.Duration_ms}
		case BW:
			fmt.Printf("BW speed %v mm/s during %v ms\n", command.Speed_mmps, command.Duration_ms)
			leftChan <- SpeedCommand{CCW, command.Speed_mmps, command.Duration_ms}
			rightChan <- SpeedCommand{CW, command.Speed_mmps, command.Duration_ms}
		/*
			case TL:
				fmt.Printf("TL %d\n", command.Iteration)
				leftChan <- SpeedCommand{CCW, command.Speed_mmps, command.Duration_ms}
				rightChan <- SpeedCommand{CCW, command.Speed_mmps, command.Duration_ms}
			case TR:
				fmt.Printf("TR %d\n", command.Iteration)
				leftChan <- SpeedCommand{CW, command.Speed_mmps, command.Duration_ms}
				rightChan <- SpeedCommand{CW, command.Speed_mmps, command.Duration_ms}
		*/
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
			steppersMove(pinStep, command.Speed_mmps, command.Duration_ms)
		case CCW:
			//fmt.Printf("CCW %d %f\n", command.Iteration, command.Pause)
			pinDirection.Low()
			steppersMove(pinStep, command.Speed_mmps, command.Duration_ms)
		default:
			fmt.Printf("processWheel2 unknown command %d\n", command.Instruction)
		}
	}
}

func steppersMove(pin rpio.Pin, speed_mmps int32, duration_ms int32) {
	dist_mm := duration_ms * speed_mmps / 1000
	iterations := float64(stepPerTurn*ticksPerStep*dist_mm) / (math.Pi * wheelSize_mm)
	pause := float64(500*duration_ms) / iterations

	steppersTicks(pin, int32(iterations), int32(pause))
}

func steppersTicks(pin rpio.Pin, iterations int32, pause int32) {
	for i := int32(0); i < iterations; i++ {
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

	//rightWhellSpeedChan <- SpeedCommand{CW, 250, 1000}
	speedCommandChan <- SpeedCommand{FW, 250, 1000}

	//smoothCommandChan <- Command{FW, stepPerTurn * ticksPerStep, 250}
	//smoothCommandChan <- Command{FW, 10, 5}

	fmt.Scanln(&input)

	disableWheels()
}
