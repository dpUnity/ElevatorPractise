package passenger

type State int

const (
	Waiting    State = iota
	InElevator State = iota
	Arrived    State = iota
)

func (s State) String() string {
	switch s {
	case InElevator:
		return "InElevator"
	case Arrived:
		return "Arrived"
	default:
		return "Waiting"
	}
}

type Passenger struct {
	ID        int
	FromFloor int
	ToFloor   int
	Status    State
}
