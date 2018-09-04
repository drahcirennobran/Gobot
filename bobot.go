package main

import (
	"fmt"
	"os"
	"time"

	"github.com/stianeikeland/go-rpio"
	//"github.com/drahcirennobran/go-rpio-mock"
)

var (
	pause [][]float64 = [][]float64{{1000, 2}, {500, 2}, {250, 4}, {100, 10}, {50, 20}, {33.3, 33}, {25, 40}, {20, 50}, {16.67, 60}, {14.29, 70}, {12.5, 80}, {11.11, 90}, {10, 100}}
)

const (
	wheelSize    = 0.1
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
	//Speed       float64
	//DurationMs  int32
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
func processSmoothCommand(smoothCmdChan chan Command, cmdChan chan Command) {
	for {
		command := <-smoothCmdChan
		switch command.Instruction {
		case FW:
			fmt.Printf("smoothCommand : FW %d %f\n", command.Iteration, command.Pause)
			accelerationCommands := SplitAcceleration(command)
			for _, splittedCommand := range accelerationCommands {
				cmdChan <- Command{FW, splittedCommand.Iteration, splittedCommand.Pause}
			}
		case BW:
			fmt.Printf("smoothCommand : BW %d\n", command.Iteration)
		case TL:
			fmt.Printf("smoothCommand : TL %d\n", command.Iteration)
		case TR:
			fmt.Printf("smoothCommand : TR %d\n", command.Iteration)
		default:
			fmt.Printf("processSmoothCommand unknown command %d\n", command.Instruction)
		}
	}
}
func processCommand(cmdChan chan Command, leftChan chan Command, rightChan chan Command) {
	for {
		command := <-cmdChan
		switch command.Instruction {
		case FW:
			fmt.Printf("FW %d ticks, pause %f\n", command.Iteration, command.Pause)
			leftChan <- Command{CW, command.Iteration, command.Pause}
			rightChan <- Command{CCW, command.Iteration, command.Pause}
		case BW:
			fmt.Printf("BW %d ticks, pause %f\n", command.Iteration, command.Pause)
			leftChan <- Command{CCW, command.Iteration, command.Pause}
			rightChan <- Command{CW, command.Iteration, command.Pause}
		case TL:
			fmt.Printf("TL %d\n", command.Iteration)
			leftChan <- Command{CCW, command.Iteration, command.Pause}
			rightChan <- Command{CCW, command.Iteration, command.Pause}
		case TR:
			fmt.Printf("TR %d\n", command.Iteration)
			leftChan <- Command{CW, command.Iteration, command.Pause}
			rightChan <- Command{CW, command.Iteration, command.Pause}
		default:
			fmt.Printf("processCommand unknown command %d\n", command.Instruction)
		}
	}
}

func processWheel(side int, c chan Command) {
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
			steppersTicks(pinStep, command.Iteration, command.Pause)
		case CCW:
			//fmt.Printf("CCW %d %f\n", command.Iteration, command.Pause)
			pinDirection.Low()
			steppersTicks(pinStep, command.Iteration, command.Pause)
		default:
			fmt.Printf("processWheel unknown command %d\n", command.Instruction)
		}
	}
}

func steppersTicks(pin rpio.Pin, iterations int, pause float64) {
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

	smoothCommandChan := make(chan Command)
	commandChan := make(chan Command)
	leftWhellTickChan := make(chan Command)
	rightWhellTickChan := make(chan Command)

	go processSmoothCommand(smoothCommandChan, commandChan)
	go processCommand(commandChan, leftWhellTickChan, rightWhellTickChan)
	go processWheel(LEFT, leftWhellTickChan)
	go processWheel(RIGHT, rightWhellTickChan)

	//rightWhellTickChan <- Command{CW, stepPerTurn * ticksPerStep, 250}

	smoothCommandChan <- Command{FW, stepPerTurn * ticksPerStep, 250}
	//smoothCommandChan <- Command{FW, 10, 5}

	fmt.Scanln(&input)

	disableWheels()
}
