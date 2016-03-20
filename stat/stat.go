package stat

import (
	"fmt"
	"github.com/gobwas/gws/stat/results"
	"github.com/gobwas/gws/stat/results/report"
	"sync"
)

type Counter interface {
	Add(v float64)
	Flush() float64
	Kind() string
}

type CounterFactory func() Counter

type Config struct {
	Factory CounterFactory
	Meta    map[string]interface{}
}

type Instance struct {
	Counter Counter
	Meta    map[string]interface{}
}

type Metric struct {
	Instances []Instance
	Tags      map[string]string
}

func (i Metric) tagsEqual(tags map[string]string) bool {
	if len(tags) != len(i.Tags) {
		return false
	}
	for k, v := range i.Tags {
		if tags[k] != v {
			return false
		}
	}
	return true
}

type Statistics struct {
	mu      sync.Mutex
	configs map[string][]*Config
	Metrics map[string][]*Metric
}

func New() *Statistics {
	return &Statistics{
		configs: make(map[string][]*Config),
		Metrics: make(map[string][]*Metric),
	}
}

func (s *Statistics) Pretty() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	report := report.New()
	for name, metrics := range s.Metrics {
		for _, metric := range metrics {
			for _, instance := range metric.Instances {
				report.AddResult(results.Result{
					Name:  name,
					Kind:  instance.Counter.Kind(),
					Value: instance.Counter.Flush(),
					Tags:  metric.Tags,
					Meta:  instance.Meta,
				})
			}
		}
	}

	return report.String()
}

func (s *Statistics) Setup(name string, config Config) (err error) {
	s.mu.Lock()
	{
		s.configs[name] = append(s.configs[name], &config)
	}
	s.mu.Unlock()
	return
}

//todo tag struct
func (s *Statistics) Increment(name string, value float64, tags map[string]string) (err error) {
	s.mu.Lock()
	{
		configs, ok := s.configs[name]
		if !ok {
			err = fmt.Errorf("metric %q has no config", name)
			s.mu.Unlock()
			return
		}

		for _, metric := range s.Metrics[name] {
			if metric.tagsEqual(tags) {
				for _, instance := range metric.Instances {
					instance.Counter.Add(value)
				}
				s.mu.Unlock()
				return
			}
		}

		metric := &Metric{Tags: tags}
		for _, config := range configs {
			instance := Instance{
				Meta:    config.Meta,
				Counter: config.Factory(),
			}
			metric.Instances = append(metric.Instances, instance)
			instance.Counter.Add(value)
		}
		s.Metrics[name] = append(s.Metrics[name], metric)
	}
	s.mu.Unlock()
	return
}
