package domains


type IDGenerator interface {
	NewID() string
}
