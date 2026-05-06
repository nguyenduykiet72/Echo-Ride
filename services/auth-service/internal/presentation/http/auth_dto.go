package http

type registerRequest struct {
	Email    string `json:"email" validate:"omitempty,email"`
	Phone    string `json:"phone" validate:"required,min=9,max=15"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required,oneof=RIDER DRIVER"`
}

type loginRequest struct {
	Phone    string `json:"phone" validate:"required"`
	Password string `json:"password" validate:"required"`
}
