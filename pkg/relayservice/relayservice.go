package relayservice

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/toni-moreno/influxdb-srelay/pkg/config"
	"github.com/toni-moreno/influxdb-srelay/pkg/relay"
)

// Service is a map of relays
type Service struct {
	relays map[string]*relay.HTTP
}

// New loads the different relays from the configuration file
func New(config *config.Config, logdir string) (*Service, error) {
	relay.SetConfig(config)
	relay.SetLogdir(logdir)
	err := relay.InitClusters()
	if err != nil {
		return nil, fmt.Errorf("Error on create Clusters: %s", err)
	}
	s := new(Service)
	s.relays = make(map[string]*relay.HTTP)

	for _, cfg := range config.HTTPConfig {
		h, err := relay.NewHTTP(cfg)
		if err != nil {
			return nil, err
		}
		if s.relays[h.Name()] != nil {
			return nil, fmt.Errorf("duplicate relay: %q", h.Name())
		}
		s.relays[h.Name()] = h
	}

	/*
		for _, cfg := range config.UDPRelays {
			u, err := relay.NewUDP(cfg)
			if err != nil {
				return nil, err
			}
			if s.relays[u.Name()] != nil {
				return nil, fmt.Errorf("duplicate relay: %q", u.Name())
			}
			s.relays[u.Name()] = u
		}*/

	return s, nil
}

// Run does run the service
// Each relay is started and the service will wait
// for them all to finish because finishing itself
func (s *Service) Run() error {
	var wg sync.WaitGroup

	var responses = make(chan error, len(s.relays))

	wg.Add(len(s.relays))

	for k := range s.relays {
		relay := s.relays[k]
		go func() {
			defer wg.Done()
			log.Printf("Running relay %s...", relay.Name())
			err := relay.Run()
			responses <- err
			if err != nil {
				log.Printf("Error running relay %q: %v", relay.Name(), err)
			}
		}()
	}

	wg.Wait()
	close(responses)
	log.Printf("Closing relays ")
	for err := range responses {
		//http.ErrServerClosed happens always when server has been closed with Shutdown/Close
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	}
	return nil
}

// Stop does stop the service by stopping each relay
func (s *Service) Stop() {
	for _, v := range s.relays {
		log.Printf("Stopping relay %s...", v.Name())
		v.Stop()
	}
}

func (s *Service) Release() {
	for _, v := range s.relays {
		log.Printf("Releasing the relay %s...", v.Name())
		v.Release()
	}
}
