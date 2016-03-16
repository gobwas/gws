package stat

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type factory func() counter

type counter interface {
	add(v float64)
	flush() float64
	name() string
}

type avg struct {
	mu    sync.Mutex
	count float64
	value float64
}

func (a *avg) add(v float64) {
	a.mu.Lock()
	{
		a.count++
		a.value += v
	}
	a.mu.Unlock()
}

func (a *avg) flush() (result float64) {
	a.mu.Lock()
	{
		if a.count == 0 {
			result = 0
		} else {
			result = a.value / a.count
		}

		a.count = 0
		a.value = 0
	}
	a.mu.Unlock()
	return
}

func (a *avg) name() string {
	return "avg"
}

type abs struct {
	mu    sync.Mutex
	value float64
}

func (a *abs) add(v float64) {
	a.mu.Lock()
	{
		a.value += v
	}
	a.mu.Unlock()
}

func (a *abs) flush() (result float64) {
	a.mu.Lock()
	{
		result = a.value
		a.value = 0
	}
	a.mu.Unlock()
	return
}

func (a *abs) name() string {
	return "abs"
}

type per struct {
	mu       sync.Mutex
	value    float64
	interval time.Duration
	stamp    time.Time
}

func (p *per) add(v float64) {
	p.mu.Lock()
	{
		p.value += v
	}
	p.mu.Unlock()
}

func (p *per) flush() (result float64) {
	p.mu.Lock()
	{
		k := float64(time.Since(p.stamp)) / float64(p.interval)
		if k == 0 {
			result = 0
		} else {
			result = p.value / k
		}

		p.value = 0
	}
	p.mu.Unlock()
	return
}

func (p *per) name() string {
	return "per " + p.interval.String()
}

type counterSetup struct {
	factory factory
	meta    map[string]interface{}
}

type counterInstance struct {
	setup   *counterSetup
	counter counter
	tags    map[string]string
}

func (i counterInstance) tagsEqual(tags map[string]string) bool {
	if len(tags) != len(i.tags) {
		return false
	}
	for k, v := range i.tags {
		if tags[k] != v {
			return false
		}
	}
	return true
}

type statistics struct {
	mu        sync.Mutex
	setups    map[string][]*counterSetup
	instances map[string][]*counterInstance
}

func newStatistics() *statistics {
	return &statistics{
		setups:    make(map[string][]*counterSetup),
		instances: make(map[string][]*counterInstance),
	}
}

const (
	titleKey   = "counter"
	titleValue = "value"
	titleKind  = "kind"
	tab        = "  "
)

type line struct {
	counter string
	kind    string
	value   string
	raw     float64
	tags    map[string]string
	meta    map[string]interface{}
}

func (l line) compare(b line) (ret int) {
	if ret = strings.Compare(l.counter, b.counter); ret != 0 {
		return
	}
	if ret = strings.Compare(l.kind, b.kind); ret != 0 {
		return
	}

	if l.raw == b.raw {
		return 0
	}
	if l.raw < b.raw {
		return -1
	}
	return +1
}

type report struct {
	lines  lines
	fields map[string]int
	meta   map[string]bool
	tags   map[string]bool
}

func newReport() *report {
	return &report{
		fields: make(map[string]int),
		meta:   make(map[string]bool),
		tags:   make(map[string]bool),
	}
}
func (p *report) field(name string, value string) {
	if cur, ok := p.fields[name]; ok {
		if l := len(value); l > cur {
			p.fields[name] = l
		}
		return
	}

	if ln, lv := len(name), len(value); lv > ln {
		p.fields[name] = lv
	} else {
		p.fields[name] = ln
	}
}
func (p *report) add(l line) {
	p.field(titleKey, l.counter)
	p.field(titleKind, l.kind)
	p.field(titleValue, l.value)
	for k, v := range l.meta {
		p.field(k, fmt.Sprintf("%v", v))
		p.meta[k] = true
	}
	for k, v := range l.tags {
		p.field(k, v)
		p.tags[k] = true
	}
	p.lines = append(p.lines, l)
}
func (p *report) count() (fields int, length int) {
	for _, v := range p.fields {
		fields++
		length += v
	}
	return
}
func (p *report) string() string {
	var buf []string

	var meta []string
	for k := range p.meta {
		meta = append(meta, k)
	}
	sort.Strings(meta)

	var tags []string
	for t := range p.tags {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	var header []string
	for _, tag := range tags {
		header = append(header, fmt.Sprintf("%-*s", p.fields[tag], tag))
	}
	header = append(header, []string{
		fmt.Sprintf("%-*s", p.fields[titleKey], titleKey),
		fmt.Sprintf("%-*s", p.fields[titleKind], titleKind),
		fmt.Sprintf("%-*s", p.fields[titleValue], titleValue),
	}...)
	for _, field := range meta {
		header = append(header, fmt.Sprintf("%-*s", p.fields[field], field))
	}
	buf = append(buf, strings.Join(header, tab))

	fields, length := p.count()
	buf = append(buf, strings.Repeat("-", length+len(tab)*fields-1))

	sort.Sort(p.lines)
	for _, l := range p.lines {
		var line []string

		for _, tag := range tags {
			var value string
			if v, ok := l.tags[tag]; ok {
				value = v
			}
			line = append(line, fmt.Sprintf("%-*s", p.fields[tag], value))
		}

		line = append(line, []string{
			fmt.Sprintf("%-*s", p.fields[titleKey], l.counter),
			fmt.Sprintf("%-*s", p.fields[titleKind], l.kind),
			fmt.Sprintf("%-*s", p.fields[titleValue], l.value),
		}...)

		for _, field := range meta {
			var value string
			if v, ok := l.meta[field]; ok {
				value = fmt.Sprintf("%s", v)
			}
			line = append(line, fmt.Sprintf("%-*s", p.fields[field], value))
		}
		buf = append(buf, strings.Join(line, tab))
	}

	return strings.Join(buf, "\n") + "\n"
}

type lines []line

func (p lines) Len() int           { return len(p) }
func (p lines) Less(i, j int) bool { return p[i].compare(p[j]) < 0 }
func (p lines) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (s *statistics) pretty() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	report := newReport()
	for id, instances := range s.instances {
		for _, instance := range instances {
			raw := instance.counter.flush()
			value := fmt.Sprintf("%.3f", raw)
			report.add(line{
				counter: id,
				kind:    instance.counter.name(),
				value:   value,
				raw:     raw,
				tags:    instance.tags,
				meta:    instance.setup.meta,
			})
		}
	}

	return report.string()
}

func (s *statistics) add(name string, setup *counterSetup) (err error) {
	s.mu.Lock()
	{
		s.setups[name] = append(s.setups[name], setup)
	}
	s.mu.Unlock()
	return
}
func (s *statistics) inc(name string, value float64, tags map[string]string) (err error) {
	s.mu.Lock()
	{
		setups, ok := s.setups[name]
		if !ok {
			err = fmt.Errorf("counter config is not exists: %q", name)
			s.mu.Unlock()
			return
		}
		need := len(setups)

		var found int
		for _, i := range s.instances[name] {
			if i.tagsEqual(tags) {
				found++
				i.counter.add(value)

				if found == need {
					break
				}
			}
		}
		if found == 0 {
			for _, setup := range setups {
				instance := &counterInstance{
					setup:   setup,
					counter: setup.factory(),
					tags:    tags,
				}
				instance.counter.add(value)
				s.instances[name] = append(s.instances[name], instance)
			}
		}
	}
	s.mu.Unlock()
	return
}
