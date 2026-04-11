package user

type User struct {
	ID        string `json:"id,omitempty" bson:"_id"`
	Username  string `json:"username" bson:"username"`
	Password  string `json:"password" bson:"password"`
	Email     string `json:"email" bson:"email"`
	Role      string `json:"role" bson:"role"`
	CreatedAt int64  `json:"created_at,omitempty" bson:"created_at"`
	IsDeleted bool   `json:"is_deleted,omitempty" bson:"is_deleted"`
	DeletedAt *int64 `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Role     string `json:"role" binding:"requried,oneof=user admin"`
}

type UpdateUserRequest struct {
	ID       string `json:"id" binding:"required"`
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type UpdatePasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	NewPassword string `json:"new_password" binding:"required"`
}
