package passenger

import (
	"elevatorpractise/config"
	"fmt"
	"math/rand"
)

func NewPassenger(id int) *Passenger {
	from := rand.Intn(config.MaxFloor) + 1
	to := from
	for to == from {
		to = rand.Intn(config.MaxFloor) + 1
	}
	p := &Passenger{
		ID:        id,
		FromFloor: from,
		ToFloor:   to,
		Status:    Waiting,
	}
	fmt.Printf("Passenger %2d created: floor %2d → floor %2d\n", p.ID, p.FromFloor, p.ToFloor)
	return p
}
