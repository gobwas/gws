package results

type Result struct {
	Name  string
	Kind  string
	Value float64
	Tags  map[string]string
	Meta  map[string]interface{}
}
