package report

import (
	"bytes"
	"fmt"
	"github.com/gobwas/gws/stat/results"
	"github.com/gobwas/gws/stat/results/sorter"
	"io"
	"sort"
	"strings"
)

const (
	titleCaption = "counter"
	titleKind    = "kind"
	titleValue   = "value"
)

type Report struct {
	results   []results.Result
	fieldSize map[string]int
	meta      map[string]bool
	tags      map[string]bool
	tab       string
}

func New() *Report {
	return &Report{
		fieldSize: make(map[string]int),
		meta:      make(map[string]bool),
		tags:      make(map[string]bool),
		tab:       "  ",
	}
}

func (p *Report) AddResult(result results.Result) {
	p.results = append(p.results, result)
	p.updateFieldsSizes(result)
}

func (p *Report) String() string {
	tags := p.getTagNamesOrder()
	meta := p.getMetaNamesOrder()

	var fields []string
	fields = append(fields, titleCaption)
	fields = append(fields, tags...)
	fields = append(fields, titleValue)
	fields = append(fields, titleKind)
	fields = append(fields, meta...)

	buf := &bytes.Buffer{}
	p.printHeader(buf, fields)
	for _, result := range p.getResultsOrdered(tags, meta) {
		var line []pair
		line = append(line, pair{titleCaption, result.Name})
		line = append(line, resultTagPairs(tags, result)...)
		line = append(line, pair{titleValue, valueToString(result.Value)})
		line = append(line, pair{titleKind, result.Kind})
		line = append(line, resultMetaPairs(meta, result)...)
		p.printLine(buf, line)
	}

	buf.WriteByte('\n')

	return buf.String()
}

func (p *Report) getResultsOrdered(tagsOrder, metaOrder []string) []results.Result {
	orderer := sorter.New(
		&sorter.CaptionComparator{},
		&sorter.TagsComparator{tagsOrder},
		&sorter.KindComparator{},
		&sorter.ValueComparator{},
	)
	return orderer.Sort(p.results)
}

func (p *Report) updateFieldsSizes(result results.Result) {
	p.updateFieldSize(titleCaption, result.Name)
	p.updateFieldSize(titleKind, result.Kind)
	p.updateFieldSize(titleValue, valueToString(result.Value))

	for k, v := range result.Meta {
		p.updateFieldSize(k, fmt.Sprintf("%v", v))
		p.meta[k] = true
	}

	for k, v := range result.Tags {
		p.updateFieldSize(k, v)
		p.tags[k] = true
	}
}

func (p *Report) updateFieldSize(name string, value string) {
	if cur, ok := p.fieldSize[name]; ok {
		if l := len(value); l > cur {
			p.fieldSize[name] = l
		}
		return
	}

	if ln, lv := len(name), len(value); lv > ln {
		p.fieldSize[name] = lv
	} else {
		p.fieldSize[name] = ln
	}
}

func (p *Report) getMetaNamesOrder() (fields []string) {
	fields = make([]string, 0, len(p.meta))
	for k := range p.meta {
		fields = append(fields, k)
	}
	sort.Strings(fields)
	return
}

func (p *Report) getTagNamesOrder() (fields []string) {
	fields = make([]string, 0, len(p.tags))
	for t := range p.tags {
		fields = append(fields, t)
	}
	sort.Strings(fields)
	return
}

func (p *Report) printHeader(w io.Writer, fields []string) (n int, err error) {
	var line []pair
	for _, f := range fields {
		line = append(line, pair{f, f})
	}
	n, err = p.printLine(w, line)
	if err != nil {
		return
	}
	w.Write([]byte(strings.Repeat("-", n-1)))
	w.Write([]byte{'\n'})
	return
}

func (p *Report) printLine(w io.Writer, line []pair) (n int, err error) {
	fieldsLen := len(line)
	if fieldsLen == 0 {
		return
	}
	var length int
	for i, pair := range line {
		size := p.fieldSize[pair.key]
		var eol string
		if i+1 == fieldsLen {
			eol = "\n"
		} else {
			eol = p.tab
		}

		n, err = fmt.Fprintf(w, "%-*s%s", size, pair.value, eol)
		if err != nil {
			return
		}
		length += n
	}

	return length, nil
}

func resultTagPairs(tags []string, result results.Result) (pairs []pair) {
	for _, key := range tags {
		var value string
		if v, ok := result.Tags[key]; ok {
			value = v
		}
		pairs = append(pairs, pair{key, value})
	}
	return
}

func resultMetaPairs(meta []string, result results.Result) (pairs []pair) {
	for _, key := range meta {
		var value string
		if v, ok := result.Meta[key]; ok {
			value = fmt.Sprintf("%s", v)
		}
		pairs = append(pairs, pair{key, value})
	}
	return
}

func valueToString(value float64) string {
	return fmt.Sprintf("%.3f", value)
}

type pair struct {
	key, value string
}
