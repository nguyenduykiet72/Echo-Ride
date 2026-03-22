package errs

var (
	ErrNotFound     = NewAppError(404, "Resource not found", "not_found")
	ErrUnauthorized = NewAppError(401, "Unauthorized", "unauthorized")
	ErrBadRequest   = NewAppError(400, "Bad request", "bad_request")
	ErrInvalidInput = NewAppError(422, "Invalid input", "invalid_input")
	ErrInternal     = NewAppError(500, "Internal server error", "internal_error")

	// Ride service specific errors
	ErrSamePickupAndDropoff = NewAppError(400, "Pickup and dropoff locations cannot be the same", "same_pickup_and_dropoff")
)
