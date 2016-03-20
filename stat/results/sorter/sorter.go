package sorter

import (
	"fmt"
	"github.com/gobwas/gws/stat/results"
	"sort"
	"strings"
)

type Sorter struct {
	comparators []Comparator
	results     []results.Result
}

func New(s ...Comparator) *Sorter {
	return &Sorter{
		comparators: s,
	}
}

func (l *Sorter) Sort(lines []results.Result) []results.Result {
	l.results = lines
	sort.Sort(l)
	return l.results
}

func (l *Sorter) Len() int {
	return len(l.results)
}

func (l *Sorter) Swap(i, j int) {
	l.results[i], l.results[j] = l.results[j], l.results[i]
}

func (l *Sorter) Less(a, b int) bool {
	lineA, lineB := l.results[a], l.results[b]
	for _, c := range l.comparators {
		switch c.compare(lineA, lineB) {
		case -1:
			return true
		case 1:
			return false
		}
	}
	return false
}

type Comparator interface {
	compare(a, b results.Result) int
}

type TagsComparator struct {
	TagNamesOrder []string
}

func (s TagsComparator) compare(a, b results.Result) int {
	return compareTags(s.TagNamesOrder, a.Tags, b.Tags)
}

type MetaComparator struct {
	MetaNamesOrder []string
}

func (s MetaComparator) compare(a, b results.Result) int {
	return compareMeta(s.MetaNamesOrder, a.Meta, b.Meta)
}

type CaptionComparator struct {
}

func (s CaptionComparator) compare(a, b results.Result) int {
	return strings.Compare(a.Name, b.Name)
}

type KindComparator struct{}

func (s KindComparator) compare(a, b results.Result) int {
	return strings.Compare(a.Kind, b.Kind)
}

type ValueComparator struct {
}

func (s ValueComparator) compare(a, b results.Result) int {
	if a.Value < b.Value {
		return -1
	}
	if a.Value > b.Value {
		return +1
	}
	return 0
}

func compareTags(priority []string, a, b map[string]string) int {
	countTagsA, countTagsB := len(a), len(b)
	if countTagsA > countTagsB {
		return -1
	}
	if countTagsA < countTagsB {
		return 1
	}
	for _, tag := range priority {
		valueA, hasA := a[tag]
		valueB, hasB := b[tag]
		if hasA && !hasB {
			return -1
		}
		if hasB && !hasA {
			return 1
		}
		if cmp := strings.Compare(valueA, valueB); cmp != 0 {
			return cmp
		}
	}
	return 0
}

func compareMeta(priority []string, a, b map[string]interface{}) int {
	countFieldsA, countFieldsB := len(a), len(b)
	if countFieldsA > countFieldsB {
		return -1
	}
	if countFieldsA < countFieldsB {
		return 1
	}
	for _, key := range priority {
		valueA, hasA := a[key]
		valueB, hasB := b[key]
		if hasA && !hasB {
			return -1
		}
		if hasB && !hasA {
			return 1
		}
		if hasA && hasB {
			if cmp := strings.Compare(fmt.Sprintf("%v", valueA), fmt.Sprintf("%v", valueB)); cmp != 0 {
				return cmp
			}
		}
	}
	return 0
}
