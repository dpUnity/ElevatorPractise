package elevator

import (
	"elevatorpractise/config"
	"elevatorpractise/passenger"
	"fmt"
)

type Direction int

const (
	IDLE Direction = iota
	UP
	DOWN
)

func (d Direction) String() string {
	switch d {
	case UP:
		return "UP"
	case DOWN:
		return "DOWN"
	default:
		return "IDLE"
	}
}

type Elevator struct {
	ID         int
	Floor      int
	Dir        Direction
	passengers []*passenger.Passenger
	targets    map[int]bool
}

func newElevator(id int) *Elevator {
	return &Elevator{
		ID:      id,
		Floor:   1,
		Dir:     IDLE,
		targets: make(map[int]bool),
	}
}

func (e *Elevator) IsFull() bool {
	return len(e.passengers) >= config.MaxCapacity
}

func (e *Elevator) IsIdle() bool {
	return e.Dir == IDLE && len(e.targets) == 0
}

// CanPickupEnRoute returns true if this non-idle, non-full elevator can pick up at floor going dir
func (e *Elevator) CanPickupEnRoute(floor int, dir Direction) bool {
	if e.IsFull() || e.IsIdle() {
		return false
	}
	if e.Dir == UP && dir == UP && floor >= e.Floor {
		return true
	}
	if e.Dir == DOWN && dir == DOWN && floor <= e.Floor {
		return true
	}
	return false
}

func (e *Elevator) StepsTo(floor int) int {
	d := floor - e.Floor
	if d < 0 {
		d = -d
	}
	return d
}

func (e *Elevator) addTarget(floor int) {
	e.targets[floor] = true
	if e.Dir == IDLE {
		if floor > e.Floor {
			e.Dir = UP
		} else if floor < e.Floor {
			e.Dir = DOWN
		}
		// floor == e.Floor: step() will detect targets[currentFloor] and stop
	}
}

// step processes one simulation tick for this elevator
func (e *Elevator) step(sys *ElevatorSystem) {
	if e.targets[e.Floor] {
		e.handleStop(sys)
		return
	}
	if len(e.targets) == 0 {
		if e.Dir != IDLE {
			e.Dir = IDLE
			fmt.Printf("[t=%3d] Elevator %d IDLE     at floor %2d\n", sys.tickCount, e.ID, e.Floor)
		}
		return
	}
	e.Floor = e.lookNextFloor()
	fmt.Printf("[t=%3d] Elevator %d moved    → floor %2d (dir=%s, onboard=%d)\n",
		sys.tickCount, e.ID, e.Floor, e.Dir, len(e.passengers))
}

// lookNextFloor returns the next floor using the LOOK algorithm
func (e *Elevator) lookNextFloor() int {
	if e.Dir == UP {
		for t := range e.targets {
			if t > e.Floor {
				return e.Floor + 1
			}
		}
		e.Dir = DOWN
		return e.Floor - 1
	}
	if e.Dir == DOWN {
		for t := range e.targets {
			if t < e.Floor {
				return e.Floor - 1
			}
		}
		e.Dir = UP
		return e.Floor + 1
	}
	// IDLE fallback: move toward nearest target
	nearest, nearestDist := -1, 999
	for t := range e.targets {
		d := t - e.Floor
		if d < 0 {
			d = -d
		}
		if d < nearestDist {
			nearestDist = d
			nearest = t
		}
	}
	if nearest > e.Floor {
		e.Dir = UP
		return e.Floor + 1
	}
	e.Dir = DOWN
	return e.Floor - 1
}

func (e *Elevator) handleStop(sys *ElevatorSystem) {
	delete(e.targets, e.Floor)
	fmt.Printf("[t=%3d] Elevator %d stopping at floor %2d (onboard=%d)\n",
		sys.tickCount, e.ID, e.Floor, len(e.passengers))

	// Drop off passengers arriving at this floor
	kept := make([]*passenger.Passenger, 0, len(e.passengers))
	for _, p := range e.passengers {
		if p.ToFloor == e.Floor {
			p.Status = passenger.Arrived
			sys.wg.Done()
			fmt.Printf("[t=%3d] Passenger %2d arrived   at floor %2d (Elevator %d)\n",
				sys.tickCount, p.ID, e.Floor, e.ID)
		} else {
			kept = append(kept, p)
		}
	}
	e.passengers = kept

	// Determine next travel direction before boarding
	nextDir := e.nextDirection()

	// Board waiting passengers going in nextDir (or any direction if elevator will be idle)
	queue := sys.floorQueues[e.Floor-1]
	remaining := make([]*passenger.Passenger, 0, len(queue))
	for _, p := range queue {
		if e.IsFull() {
			remaining = append(remaining, p)
			continue
		}
		pDir := directionOf(p.FromFloor, p.ToFloor)
		if nextDir == IDLE || pDir == nextDir {
			e.passengers = append(e.passengers, p)
			p.Status = passenger.InElevator
			e.targets[p.ToFloor] = true
			fmt.Printf("[t=%3d] Passenger %2d boarded   Elevator %d at floor %2d → %2d\n",
				sys.tickCount, p.ID, e.ID, p.FromFloor, p.ToFloor)
		} else {
			remaining = append(remaining, p)
		}
	}
	sys.floorQueues[e.Floor-1] = remaining

	e.Dir = e.nextDirection()
	if e.Dir == IDLE {
		fmt.Printf("[t=%3d] Elevator %d IDLE     at floor %2d\n", sys.tickCount, e.ID, e.Floor)
	}
}

func (e *Elevator) nextDirection() Direction {
	hasAbove, hasBelow := false, false
	for t := range e.targets {
		if t > e.Floor {
			hasAbove = true
		}
		if t < e.Floor {
			hasBelow = true
		}
	}
	switch e.Dir {
	case UP:
		if hasAbove {
			return UP
		}
		if hasBelow {
			return DOWN
		}
	case DOWN:
		if hasBelow {
			return DOWN
		}
		if hasAbove {
			return UP
		}
	default:
		if hasAbove {
			return UP
		}
		if hasBelow {
			return DOWN
		}
	}
	return IDLE
}

func directionOf(from, to int) Direction {
	if to > from {
		return UP
	}
	return DOWN
}
