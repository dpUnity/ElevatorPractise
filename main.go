package main

import (
	"elevatorpractise/config"
	"elevatorpractise/elevator"
	"elevatorpractise/passenger"
	"fmt"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup
	sys := elevator.NewElevatorSystem(&wg)
	go sys.Run()

	ticker := time.NewTicker(config.TickDuration)
	defer ticker.Stop()
	start := time.Now()

	for i := 1; i <= config.TotalPassengers; i++ {
		<-ticker.C
		p := passenger.NewPassenger(i)
		wg.Add(1)
		sys.PassengerCh <- p
	}
	close(sys.PassengerCh)

	wg.Wait()
	elapsed := time.Since(start)
	fmt.Printf("=== All %d passengers served in %.1f seconds ===\n",
		config.TotalPassengers, elapsed.Seconds())
}
