package speeddaemon

// A Camera represents a speed camera.
//
// Each camera is on a specific road, at a specific location, and has a specific speed limit.
// Each camera provides this information when it connects to the server.
// Cameras report each number plate that they observe, along with the timestamp that they observed it.
// Timestamps are exactly the same as [Unix timestamps] (counting seconds since 1st of January 1970), except that they are unsigned.
//
// [Unix timestamps]: https://en.wikipedia.org/wiki/Unix_time
type Camera struct {
	Road  uint16
	Mile  uint16
	Limit uint16
}

// A TicketDispatcher is responsible for some number of roads.
//
// When the server finds that a car was detected at 2 points on the same road with an average speed in excess of the
// speed limit (speed = distance / time), it will find the responsible ticket dispatcher and send it a ticket for the
// offending car, so that the ticket dispatcher can perform the necessary legal rituals.
type TicketDispatcher struct {
	Roads []uint16
}

// A Road in the network is identified by a number from 0 to 65535.
//
// A single road has the same speed limit at every point on the road.
// Positions on the roads are identified by the number of miles from the start of the road.
// Remarkably, all speed cameras are positioned at exact integer numbers of miles from the start of the road.
type Road struct{}

// A Car has a specific number plate represented as an uppercase alphanumeric string.
type Car string

type CameraRecord struct {
}
