package service

import (
	"jfrog-credential-provider/internal/logger"
	"net/http"
)

type Service struct {
	Client *http.Client
	Logger logger.Logger
}

func NewService(client *http.Client, logger logger.Logger) *Service {
	return &Service{
		Client: client,
		Logger: logger,
	}
}
