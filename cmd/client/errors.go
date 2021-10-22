package main

import "errors"

var (
	errCommandIncorrect   = errors.New("command malformed or incomplete")
	errKeyOrValueMissing  = errors.New("key or value missing")
	errCommandNotProvided = errors.New("command not provided")
)
