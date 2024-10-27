package supervisor

// Define a Message type
type Message struct {
	Request  string      // Request type
	Payload  interface{} // Data payload for the message
	Response chan string // Channel for sending responses (optional)
}
