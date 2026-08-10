package elevator

import (
	"elevatorpractise/config"
	"elevatorpractise/passenger"
	"fmt"
	"sync"
	"time"
)

type ElevatorSystem struct {
	elevators   []*Elevator
	floorQueues [][]*passenger.Passenger
	PassengerCh chan *passenger.Passenger
	wg          *sync.WaitGroup
	mu          sync.Mutex
	tickCount   int
}

func NewElevatorSystem(wg *sync.WaitGroup) *ElevatorSystem {
	sys := &ElevatorSystem{
		elevators:   make([]*Elevator, config.NumElevators),
		floorQueues: make([][]*passenger.Passenger, config.MaxFloor),
		PassengerCh: make(chan *passenger.Passenger, config.PassengerChBuf),
		wg:          wg,
	}
	for i := 0; i < config.NumElevators; i++ {
		sys.elevators[i] = newElevator(i + 1)
		fmt.Printf("Elevator %d initialized at floor 1\n", i+1)
	}
	for i := range sys.floorQueues {
		sys.floorQueues[i] = make([]*passenger.Passenger, 0)
	}
	return sys
}

func (s *ElevatorSystem) Run() {
	ticker := time.NewTicker(config.TickDuration)
	defer ticker.Stop()
	channelClosed := false
	for {
		<-ticker.C
		s.tickCount++
		// Drain all pending passengers before stepping elevators
		if !channelClosed {
		drain:
			for {
				select {
				case p, ok := <-s.PassengerCh:
					if !ok {
						channelClosed = true
						break drain
					}
					s.dispatch(p)
				default:
					break drain
				}
			}
		}
		for _, e := range s.elevators {
			e.step(s)
		}
		s.checkQueues()

		// Check if all passengers have been served and channel is closed
		if channelClosed && s.allServed() {
			fmt.Printf("\n[t=%3d] All passengers served.\n", s.tickCount)
			return
		}
	}
}

func (s *ElevatorSystem) allServed() bool {
	// Check if any elevator has passengers or targets
	for _, e := range s.elevators {
		if len(e.passengers) > 0 || len(e.targets) > 0 {
			return false
		}
	}
	// Check if any floor has waiting passengers
	for _, queue := range s.floorQueues {
		if len(queue) > 0 {
			return false
		}
	}
	return true
}

func (s *ElevatorSystem) dispatch(p *passenger.Passenger) {
	s.floorQueues[p.FromFloor-1] = append(s.floorQueues[p.FromFloor-1], p)
	fmt.Printf("[t=%3d] Passenger %2d queued    at floor %2d → %2d\n",
		s.tickCount, p.ID, p.FromFloor, p.ToFloor)
	if !s.assignToElevator(p) {
		fmt.Printf("[t=%3d] Passenger %2d waiting   at floor %2d (no elevator available)\n",
			s.tickCount, p.ID, p.FromFloor)
	}
}

func (s *ElevatorSystem) assignToElevator(p *passenger.Passenger) bool {
	pDir := directionOf(p.FromFloor, p.ToFloor)

	// 1. Closest idle elevator
	var best *Elevator
	bestDist := 999
	for _, e := range s.elevators {
		if e.IsIdle() {
			if d := e.StepsTo(p.FromFloor); d < bestDist {
				bestDist = d
				best = e
			}
		}
	}
	if best != nil {
		best.addTarget(p.FromFloor)
		fmt.Printf("[t=%3d] Passenger %2d → Elevator %d (idle,     dist=%d)\n",
			s.tickCount, p.ID, best.ID, bestDist)
		return true
	}

	// 2. Closest en-route elevator (same direction, not full, floor ahead)
	bestDist = 999
	for _, e := range s.elevators {
		if e.CanPickupEnRoute(p.FromFloor, pDir) {
			if d := e.StepsTo(p.FromFloor); d < bestDist {
				bestDist = d
				best = e
			}
		}
	}
	if best != nil {
		best.addTarget(p.FromFloor)
		fmt.Printf("[t=%3d] Passenger %2d → Elevator %d (en route, dist=%d)\n",
			s.tickCount, p.ID, best.ID, bestDist)
		return true
	}

	return false
}

func (s *ElevatorSystem) checkQueues() {
	// Build the set of floors already targeted by an elevator
	covered := make(map[int]bool)
	for _, e := range s.elevators {
		for f := range e.targets {
			covered[f] = true
		}
	}
	for floor := 1; floor <= config.MaxFloor; floor++ {
		if len(s.floorQueues[floor-1]) > 0 && !covered[floor] {
			s.assignToElevator(s.floorQueues[floor-1][0])
		}
	}
}
