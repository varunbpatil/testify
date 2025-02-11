package suite_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/varunbpatil/testify/suite"
)

// ParallelSuite is the test suite.
// This data can be initialized in [ParallelSuite.SetupSuite].
type ParallelSuite struct {
	// Data common to all tests.
	GlobalData string
}

// PerTestData holds data unique to each test.
// This data can be initialized in [ParallelSuite.SetupTest]
// and/or [ParallelSuite.SetupSubTest].
type PerTestData struct {
	PerTestData string
}

// Suite level setup and teardown.
func (s *ParallelSuite) SetupSuite(t *suite.T[PerTestData]) {
	t.Log("SetupSuite:", t.Name())
	s.GlobalData = "<< Global Data >>"
}

func (s *ParallelSuite) TearDownSuite(t *suite.T[PerTestData]) {
	t.Log("TearDownSuite:", t.Name(), "global data:", s.GlobalData)
}

// Test level setup and teardown.
func (s *ParallelSuite) SetupTest(t *suite.T[PerTestData]) {
	t.Log("SetupTest:", t.Name())
	t.D().PerTestData = fmt.Sprintf("<< Per-Test Data for %s>>", t.Name())
}

func (s *ParallelSuite) TearDownTest(t *suite.T[PerTestData]) {
	t.Log("TearDownTest:", t.Name(), "per-test data:", t.D().PerTestData)
}

func (s *ParallelSuite) BeforeTest(t *suite.T[PerTestData], suiteName, testName string) {
	t.Log("BeforeTest:", t.Name())
}

func (s *ParallelSuite) AfterTest(t *suite.T[PerTestData], suiteName, testName string) {
	t.Log("AfterTest:", t.Name())
}

// Subtest level setup and teardown.
func (s *ParallelSuite) SetupSubTest(t *suite.T[PerTestData]) {
	t.Log(fmt.Sprintf("SetupSubTest: %s", t.Name()))
	t.D().PerTestData = fmt.Sprintf("<< Sub-Test Data for %s>>", t.Name())
}

func (s *ParallelSuite) TearDownSubTest(t *suite.T[PerTestData]) {
	t.Log("TearDownSubTest:", t.Name(), "sub-test data:", t.D().PerTestData)
}

func (s *ParallelSuite) TestOne(t *suite.T[PerTestData]) {
	t.Parallel()
	t.Log("started running:", t.Name(), "with data:", t.D().PerTestData)

	for _, v := range []string{"sub1", "sub2", "sub3"} {
		t.Run(v, func(t *suite.T[PerTestData]) {
			t.Parallel()

			r := rand.Intn(3)
			t.Log("started running:", t.Name(), "with data:", t.D().PerTestData)
			time.Sleep(time.Duration(r) * time.Second)
			t.Log("stopped running:", t.Name())
		})
	}
}

func (s *ParallelSuite) TestTwo(t *suite.T[PerTestData]) {
	t.Parallel()
	t.Log("started running:", t.Name(), "with data:", t.D().PerTestData)

	for _, v := range []string{"sub1", "sub2", "sub3"} {
		t.Run(v, func(t *suite.T[PerTestData]) {
			t.Parallel()

			r := rand.Intn(3)
			t.Log("started running:", t.Name(), "with data:", t.D().PerTestData)
			time.Sleep(time.Duration(r) * time.Second)
			t.Log("stopped running:", t.Name())
		})
	}
}

// TestSuiteParallel is the main entrypoint for the test.
func TestSuiteParallel(t *testing.T) {
	suite.Run[PerTestData](t, new(ParallelSuite))
}
