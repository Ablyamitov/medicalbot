package entities

type TestResult struct {
	SessionID       string   `json:"session_id"`
	TestType        TestType `json:"test_type"`
	Score           int      `json:"score"`
	Interpretation  string   `json:"interpretation"`
	Recommendations []string `json:"recommendations"`
}
