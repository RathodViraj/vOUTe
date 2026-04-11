package user

import (
	"context"
	"time"
	"voute/pkg/utils"
)

type UserService interface {
	CreateUser(ctx context.Context, name, email, password, role string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUsersByUsername(ctx context.Context, username string, skip, take int) ([]*User, error)
	UpdateUser(ctx context.Context, name, email, id string) error
	DeleteUser(ctx context.Context, id string) error
	UpdatePassword(ctx context.Context, email, password string) error
}

type userService struct {
	userRepo UserRepository
}

func NewUserService(userRepo UserRepository) UserService {
	return &userService{
		userRepo: userRepo,
	}
}

func (s *userService) CreateUser(ctx context.Context, name, email, password, role string) (*User, error) {
	hashPwd, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &User{
		Username:  name,
		Email:     email,
		Password:  hashPwd,
		Role:      "user",
		CreatedAt: time.Now().Unix(),
	}
	err = s.userRepo.CreateUser(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (s *userService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return s.userRepo.GetUserByEmail(ctx, email)
}

func (s *userService) GetUserByID(ctx context.Context, id string) (*User, error) {
	return s.userRepo.GetUserByID(ctx, id)
}

func (s *userService) GetUsersByUsername(ctx context.Context, username string, skip, take int) ([]*User, error) {
	if skip < 0 {
		skip = 0
	}
	if take <= 0 || take > 100 {
		take = 10
	}

	return s.userRepo.GetUsersByUsername(ctx, username, skip, take)
}

func (s *userService) UpdateUser(ctx context.Context, name, email, id string) error {
	return s.userRepo.UpdateUser(ctx, name, email, id)
}

func (s *userService) DeleteUser(ctx context.Context, id string) error {
	return s.userRepo.DeleteUser(ctx, id)
}

func (s *userService) UpdatePassword(ctx context.Context, email, password string) error {
	hashPwd, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	return s.userRepo.UpdatePassword(ctx, email, hashPwd)
}
