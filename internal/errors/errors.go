package myErrors

import "errors"

var (
	ErrUserNotFound    = errors.New("invalid username")
	ErrInvalidPassword = errors.New("invalid password")
	ErrUserExists      = errors.New("user already exists")
	ErrRoomNotFound    = errors.New("room not found")
	ErrUnauthorized    = errors.New("unauthorized")
)
