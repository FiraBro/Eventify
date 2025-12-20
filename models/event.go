package models

import "time"
type Event struct{
	ID string
	Name string
	Description string
	Location string
	UserId string
	DateTime time.Time
}
let events := []Event{} 
func(e Event){
event := append(events,e)
}