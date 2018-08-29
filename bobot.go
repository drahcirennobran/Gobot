package main

import (
	"fmt"
	"os"
	"time"

	//"github.com/stianeikeland/go-rpio"
	"github.com/drahcirennobran/go-rpio-mock"
	"github.com/drahcirennobran/queue"
)

var (
	pause             [][]float64 = [][]float64{{1000, 2}, {500, 2}, {250, 4}, {100, 10}, {50, 20}, {33.3, 33}, {25, 40}, {20, 50}, {16.67, 60}, {14.29, 70}, {12.5, 80}, {11.11, 90}, {10, 100}}
	pinStepRight                  = rpio.Pin(16) //36
	pinDirectionRight             = rpio.Pin(20) //38
	pinDisableRight               = rpio.Pin(21) //40
	pinStepLeft                   = rpio.Pin(13) //33
	pinDirectionLeft              = rpio.Pin(19) //35
	pinDisableLeft                = rpio.Pin(26) //37

	pinVoid = rpio.Pin(31)
)

const (
	LEFT  int = 1
	RIGHT int = 2
	CW    int = 3
	CCW   int = 4
	FW    int = 5
	BW    int = 6
	TL    int = 7
	TR    int = 8
	ACCFW int = 9
	DECFW int = 10
	ACCBW int = 11
	DECBW int = 12
)

type Command struct {
	Instruction int
	Iteration   int
	Pause       float64
}

func SplitAcceleration(command queue.Command) []queue.Command {
	splittedCommands := make([]queue.Command, 0)
	for i, totalTicks := 0, 0; pause[i][0] >= command.Pause && i < len(pause) && totalTicks < command.Iteration; i, totalTicks = i+1, totalTicks+int(pause[i][1]) {
		splittedCommands = append(splittedCommands, queue.Command{0, int(pause[i][1]), pause[i][0]})
		fmt.Printf("i=%d ; %d pause de %f\n", i, int(pause[i][1]), pause[i][0])
	}
	return splittedCommands
}
func processSmoothCommand(smoothCmdChan chan queue.Command, cmdChan chan queue.Command) {
	for {
		command := <-smoothCmdChan
		switch command.Instruction {
		case ACCFW:
			fmt.Printf("ACCFW %d %f\n", command.Iteration, command.Pause)
			accelerationCommands := SplitAcceleration(command)
			println("accelerationCommands size %d", len(accelerationCommands))
			for _, splittedCommand := range accelerationCommands {
				cmdChan <- queue.Command{FW, splittedCommand.Iteration, splittedCommand.Pause}
			}
		case DECFW:
			fmt.Printf("DECFW %d\n", command.Iteration)
		case ACCBW:
			fmt.Printf("ACCBW %d\n", command.Iteration)
		case DECBW:
			fmt.Printf("DECBW %d\n", command.Iteration)
		default:
			fmt.Printf("unknown command %d\n", command.Instruction)
		}
	}
}
func processCommand(cmdChan chan queue.Command, leftChan chan queue.Command, rightChan chan queue.Command) {
	for {
		command := <-cmdChan
		switch command.Instruction {
		case FW:
			fmt.Printf("FW %d ticks, pause %f\n", command.Iteration, command.Pause)
			leftChan <- queue.Command{CW, command.Iteration, command.Pause}
			rightChan <- queue.Command{CCW, command.Iteration, command.Pause}
		case BW:
			fmt.Printf("BW %d ticks, pause %f\n", command.Iteration, command.Pause)
			leftChan <- queue.Command{CCW, command.Iteration, command.Pause}
			rightChan <- queue.Command{CW, command.Iteration, command.Pause}
		case TL:
			fmt.Printf("TL %d\n", command.Iteration)
			leftChan <- queue.Command{CCW, command.Iteration, command.Pause}
			rightChan <- queue.Command{CCW, command.Iteration, command.Pause}
		case TR:
			fmt.Printf("TR %d\n", command.Iteration)
			leftChan <- queue.Command{CW, command.Iteration, command.Pause}
			rightChan <- queue.Command{CW, command.Iteration, command.Pause}
		default:
			fmt.Printf("unknown command %d\n", command.Instruction)
		}
	}
}

func processWheel(side int, c chan queue.Command) {
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
			fmt.Printf("unknown command %d\n", command.Instruction)
		}
		steppersTicks(pinStep, command.Iteration, command.Pause)
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
	pinStepRight.Output()
	pinDirectionRight.Output()
	pinDisableRight.Output()
	pinStepLeft.Output()
	pinDirectionLeft.Output()
	pinDisableLeft.Output()

	pinDisableRight.Low()
	pinDisableLeft.Low()

	smoothCommandChan := make(chan queue.Command)
	commandChan := make(chan queue.Command)
	leftWhellChan := make(chan queue.Command)
	rightWhellChan := make(chan queue.Command)

	go processSmoothCommand(smoothCommandChan, commandChan)
	go processCommand(commandChan, leftWhellChan, rightWhellChan)
	go processWheel(LEFT, leftWhellChan)
	go processWheel(RIGHT, rightWhellChan)

	smoothCommandChan <- queue.Command{ACCFW, 10, 50}
	//smoothCommandChan <- queue.Command{ACCFW, 10, 5}
	fmt.Scanln(&input)

	pinDisableRight.High()
	pinDisableLeft.High()
}
