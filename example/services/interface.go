package services

import "context"

type Service interface {
	Healthy(context.Context) bool
	Run(context.Context) error
}
