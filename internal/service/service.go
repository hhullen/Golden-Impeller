package service

type Broker interface {
}

type Strategy interface {
}

type Service struct {
	broker Broker
}

func NewService(b Broker) *Service {
	return &Service{
		broker: b,
	}
}
