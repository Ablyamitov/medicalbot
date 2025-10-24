package entities

type TestType string

const (
	TestTypeAMS  TestType = "AMS"
	TestTypeMIEF TestType = "MIEF"
	TestTypeIPSS TestType = "IPSS"
)

type Test struct {
	ID          int64      `json:"id"`
	Type        TestType   `json:"type"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Disclaimer  string     `json:"disclaimer"`
	Questions   []Question `json:"questions"`
}
