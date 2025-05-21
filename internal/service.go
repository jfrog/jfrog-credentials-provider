package service

import (
	"log"
	"net/http"
)

type Service struct {
	Client *http.Client
	Logger *log.Logger
}

func NewService(client *http.Client, logger *log.Logger) *Service {
	return &Service{
		Client: client,
		Logger: logger,
	}
}
