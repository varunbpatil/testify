// This is basically the same as stretchr/testify suite tests defined at
// https://github.com/stretchr/testify/blob/master/suite/suite_test.go
// but modified to work with the parallel testify suite implemented by
// this package.
//
// IMPORTANT: When you run the tests in this file, especially in verbose mode, you will see several
// tests/subtests failing/panicking. This is INTENTIONAL because these tests are testing the
// implementation/internals of the parallel test suite. However, the overall result of the
// `go test ./...` command should be OK.
package suite_test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/varunbpatil/testify/suite"
)

var allTestsFilter = func(_, _ string) (bool, error) { return true, nil }

// ParallelSuite is intended to be a fully functional example of a parallel testing suite.
// To make this look more like a real world example, all tests/sub-tests in this suite are
// parallel. It also shows how global and per-test data can be used. The suite also verifies
// the correct order of setup and teardown of suite/test/sub-test.
type ParallelSuite struct {
	*suite.Suite[ParallelSuite, GlobalData]

	// All the per-test data is stored here.
	// This will be unique to each test/sub-test.
	PerTestData string
}

type GlobalData struct {
	// All the global data is stored here...
	GlobalData           string
	SetupTearDownTracker *SetupTearDownTracker
}

type SetupTearDownTracker struct {
	sync.Mutex
	SetupTearDownTracker []string
}

func (t *SetupTearDownTracker) append(s string) {
	t.Lock()
	defer t.Unlock()
	t.SetupTearDownTracker = append(t.SetupTearDownTracker, s)
}

// Suite level setup and teardown.
func (s *ParallelSuite) SetupSuite() {
	s.Log("SetupSuite:", s.Name())
	s.G().GlobalData = "[G]"
	s.G().SetupTearDownTracker = &SetupTearDownTracker{}
	s.G().SetupTearDownTracker.append(fmt.Sprintf(">%s", s.Name()))
}

func (s *ParallelSuite) TearDownSuite() {
	s.Log("TearDownSuite:", s.Name(), s.G().GlobalData)
	s.G().SetupTearDownTracker.append(fmt.Sprintf("<%s", s.Name()))

	// Verify the setup and teardown order of the whole suite. Since the test names look like
	// filesystem paths, we can use the filesystem to verify the order.
	//
	// Whenever we encounter a ">", we create a directory, and whenever we encounter a "<", we
	// remove the directory. This will create a tree of directories with the suite as the root,
	// the tests in the suite as level 1 subdirectories, and the subtests as level 2
	// subdirectories.
	//
	// If the order is incorrect, we will get an error while trying to create a subdirectory in
	// a parent directory that doesn't exist or while trying to remove a directory that is not
	// empty.
	s.Log("Verifying setup and teardown order...")
	s.Log("SetupTearDownTracker:", s.G().SetupTearDownTracker.SetupTearDownTracker)
	s.Require().Equal(20, len(s.G().SetupTearDownTracker.SetupTearDownTracker))
	for _, tests := range s.G().SetupTearDownTracker.SetupTearDownTracker {
		if strings.HasPrefix(tests, ">") {
			path := tests[1:]
			err := os.Mkdir(path, 0755)
			if err != nil {
				s.Fatalf("Error creating directory: %v. err: %s", path, err)
			}
		} else if strings.HasPrefix(tests, "<") {
			path := tests[1:]
			err := os.Remove(path)
			if err != nil {
				s.Fatalf("Error removing directory: %s. err: %v", path, err)
			}
		}
	}
	// Verify that there is no directory named `TestSuiteParallel` in the current directory
	// because if the order of setup and teardown was correct, the directory should have been
	// removed.
	_, err := os.Stat("TestSuiteParallel")
	if err == nil {
		s.Fatalf("Directory `TestSuiteParallel` shouldn't exist")
	}

}

// Test level setup and teardown.
func (s *ParallelSuite) SetupTest() {
	s.Log("SetupTest:", s.Name())
	s.PerTestData = fmt.Sprintf("{%s}", s.Name())
	s.G().SetupTearDownTracker.append(fmt.Sprintf(">%s", s.Name()))
}

func (s *ParallelSuite) TearDownTest() {
	s.Log("TearDownTest:", s.Name(), s.PerTestData)
	s.G().SetupTearDownTracker.append(fmt.Sprintf("<%s", s.Name()))
}

func (s *ParallelSuite) BeforeTest(suiteName, testName string) {
	s.Log("BeforeTest:", s.Name())
}

func (s *ParallelSuite) AfterTest(suiteName, testName string) {
	s.Log("AfterTest:", s.Name())
}

// Subtest level setup and teardown.
func (s *ParallelSuite) SetupSubTest() {
	s.Log("SetupSubTest:", s.Name())
	s.PerTestData = fmt.Sprintf("(%s)", s.Name())
	s.G().SetupTearDownTracker.append(fmt.Sprintf(">%s", s.Name()))
}

func (s *ParallelSuite) TearDownSubTest() {
	s.Log("TearDownSubTest:", s.Name(), s.PerTestData)
	s.G().SetupTearDownTracker.append(fmt.Sprintf("<%s", s.Name()))
}

// HandleStats is called when the test suite is finished.
func (s *ParallelSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
	spew.Dump(stats)
}

func (s *ParallelSuite) TestOne() {
	s.Parallel()
	s.Log("started running:", s.Name(), s.G().GlobalData, s.PerTestData)

	for _, v := range []string{"sub1", "sub2", "sub3"} {
		s.Run(v, func(s *ParallelSuite) {
			s.Parallel()

			r := rand.Intn(3)
			s.Log("started running:", s.Name(), s.G().GlobalData, s.PerTestData)
			time.Sleep(time.Duration(r) * time.Second)
			s.Log("stopped running:", s.Name())
		})
	}
}

func (s *ParallelSuite) TestTwo() {
	s.Parallel()
	s.Log("started running:", s.Name(), s.G().GlobalData, s.PerTestData)

	for _, v := range []string{"sub1", "sub2", "sub3"} {
		s.Run(v, func(s *ParallelSuite) {
			s.Parallel()

			r := rand.Intn(3)
			s.Log("started running:", s.Name(), s.G().GlobalData, s.PerTestData)
			time.Sleep(time.Duration(r) * time.Second)
			s.Log("stopped running:", s.Name())
		})
	}
}

// This test will be skipped.
func (suite *ParallelSuite) TestSkip() {
	suite.T().Skip()
}

// TestSuiteParallel is the main entrypoint for the test.
func TestSuiteParallel(t *testing.T) {
	t.Parallel()
	suite.Run[ParallelSuite, GlobalData](t)
}

// SuiteRequireTwice is intended to test the usage of suite.Require in two
// different tests
type SuiteRequireTwice struct {
	*suite.Suite[SuiteRequireTwice, SuiteRequireTwiceGlobalData]
}

type SuiteRequireTwiceGlobalData struct{}

// TestSuiteRequireTwice checks for regressions of issue #149 where
// suite.requirements was not initialized in suite.SetT()
// A regression would result on these tests panicking rather than failing.
func TestSuiteRequireTwice(t *testing.T) {
	ok := testing.RunTests(
		allTestsFilter,
		[]testing.InternalTest{{
			Name: t.Name() + "/SuiteRequireTwice",
			F: func(t *testing.T) {
				suite.Run[SuiteRequireTwice, SuiteRequireTwiceGlobalData](t)
			},
		}},
	)
	assert.False(t, ok)
}

func (s *SuiteRequireTwice) TestRequireOne() {
	r := s.Require()
	r.Equal(1, 2)
}

func (s *SuiteRequireTwice) TestRequireTwo() {
	r := s.Require()
	r.Equal(1, 2)
}

// panickingSuite is intended to test that the test suite recovers from panics
// in setup/teardown methods as well as panics in the tests/subtests themselves.
type panickingSuite struct {
	*suite.Suite[panickingSuite, panickingSuiteGlobalData]
}

type panickingSuiteGlobalData struct{}

func (s *panickingSuite) SetupSuite() {
	if strings.Contains(s.Name(), "/InSetupSuite") {
		panic("oops in setup suite")
	}
}

func (s *panickingSuite) SetupTest() {
	if strings.Contains(s.Name(), "/InSetupTest") {
		panic("oops in setup test")
	}
}

func (s *panickingSuite) BeforeTest(_, _ string) {
	if strings.Contains(s.Name(), "/InBeforeTest") {
		panic("oops in before test")
	}
}

func (s *panickingSuite) SetupSubTest() {
	if strings.Contains(s.Name(), "/InSetupSubTest") {
		panic("oops in setup subtest")
	}
}

func (s *panickingSuite) Test() {
	if strings.Contains(s.Name(), "/InTest") {
		panic("oops in test")
	}

	s.Run("SubTest", func(s *panickingSuite) {
		if strings.Contains(s.Name(), "/InSubTest") {
			panic("oops in subtest")
		}
	})
}

func (s *panickingSuite) TearDownSubTest() {
	if strings.Contains(s.Name(), "/InTearDownSubTest") {
		panic("oops in tear down subtest")
	}
}

func (s *panickingSuite) AfterTest(_, _ string) {
	if strings.Contains(s.Name(), "/InAfterTest") {
		panic("oops in after test")
	}
}

func (s *panickingSuite) TearDownTest() {
	if strings.Contains(s.Name(), "/InTearDownTest") {
		panic("oops in tear down test")
	}
}

func (s *panickingSuite) TearDownSuite() {
	if strings.Contains(s.Name(), "/InTearDownSuite") {
		panic("oops in tear down suite")
	}
}

func (s *panickingSuite) HandleStats(suiteName string, stats *suite.SuiteInformation) {
	if strings.Contains(s.Name(), "/InHandleStats") {
		panic("oops in handle stats")
	}
}

func TestSuiteRecoverPanic(t *testing.T) {
	ok := true
	panickingTests := []testing.InternalTest{
		{
			Name: t.Name() + "/InSetupSuite",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InSetupTest",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InBeforeTest",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InSetupSubTest",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InTest",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InSubTest",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InTearDownSubTest",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InAfterTest",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InTearDownTest",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InTearDownSuite",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
		{
			Name: t.Name() + "/InHandleStats",
			F:    func(t *testing.T) { suite.Run[panickingSuite, panickingSuiteGlobalData](t) },
		},
	}

	require.NotPanics(t, func() {
		ok = testing.RunTests(allTestsFilter, panickingTests)
	})

	assert.False(t, ok)
}

// This suite has no Test... methods. It's setup and teardown must be skipped.
type SuiteSetupSkipTester struct {
	*suite.Suite[SuiteSetupSkipTester, SuiteSetupSkipTesterGlobalData]
}

type SuiteSetupSkipTesterGlobalData struct{}

func (s *SuiteSetupSkipTester) SetupSuite() {
	panic("should never be called because there are no tests in this test suite")
}

func (s *SuiteSetupSkipTester) NonTestMethod() {

}

func (s *SuiteSetupSkipTester) TearDownSuite() {
	panic("should never be called because there are no tests in this test suite")
}

func TestSkippingSuiteSetup(t *testing.T) {
	suite.Run[SuiteSetupSkipTester, SuiteSetupSkipTesterGlobalData](t)
}

type SuiteLoggingTester struct {
	*suite.Suite[SuiteLoggingTester, SuiteLoggingTesterGlobalData]
}

type SuiteLoggingTesterGlobalData struct{}

func (s *SuiteLoggingTester) TestLoggingPass() {
	s.T().Log("TESTLOGPASS")
}

func (s *SuiteLoggingTester) TestLoggingFail() {
	s.T().Log("TESTLOGFAIL")
	assert.NotNil(s.T(), nil) // expected to fail
}

type StdoutCapture struct {
	oldStdout *os.File
	readPipe  *os.File
}

func (sc *StdoutCapture) StartCapture() {
	sc.oldStdout = os.Stdout
	sc.readPipe, os.Stdout, _ = os.Pipe()
}

func (sc *StdoutCapture) StopCapture() (string, error) {
	if sc.oldStdout == nil || sc.readPipe == nil {
		return "", errors.New("StartCapture not called before StopCapture")
	}
	os.Stdout.Close()
	os.Stdout = sc.oldStdout
	bytes, err := io.ReadAll(sc.readPipe)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func TestSuiteLogging(t *testing.T) {
	capture := StdoutCapture{}
	internalTest := testing.InternalTest{
		Name: t.Name() + "/SuiteLoggingTester",
		F: func(subT *testing.T) {
			suite.Run[SuiteLoggingTester, SuiteLoggingTesterGlobalData](subT)
		},
	}
	capture.StartCapture()
	testing.RunTests(allTestsFilter, []testing.InternalTest{internalTest})
	output, err := capture.StopCapture()
	require.NoError(t, err, "Got an error trying to capture stdout and stderr!")
	require.NotEmpty(t, output, "output content must not be empty")

	// Failed tests' output is always printed
	assert.Contains(t, output, "TESTLOGFAIL")

	if testing.Verbose() {
		// In verbose mode, output from successful tests is also printed
		assert.Contains(t, output, "TESTLOGPASS")
	} else {
		assert.NotContains(t, output, "TESTLOGPASS")
	}
}

type suiteWithStats struct {
	*suite.Suite[suiteWithStats, suiteWithStatsGlobalData]
}

var (
	wasCalled bool
	stats     *suite.SuiteInformation
)

type suiteWithStatsGlobalData struct{}

func (s *suiteWithStats) HandleStats(suiteName string, st *suite.SuiteInformation) {
	wasCalled = true
	stats = st
}

func (s *suiteWithStats) TestSomething() {
	s.Equal(1, 1)
}

func (s *suiteWithStats) TestPanic() {
	panic("oops")
}

func TestSuiteWithStats(t *testing.T) {
	suiteSuccess := testing.RunTests(allTestsFilter, []testing.InternalTest{
		{
			Name: t.Name() + "/suiteWithStats",
			F: func(t *testing.T) {
				suite.Run[suiteWithStats, suiteWithStatsGlobalData](t)
			},
		},
	})
	require.False(t, suiteSuccess, "suiteWithStats should report test failure because of panic in TestPanic")

	assert.True(t, wasCalled)
	assert.NotZero(t, stats.Start)
	assert.NotZero(t, stats.End)
	assert.False(t, stats.Passed())

	testStats := stats.TestStats

	assert.NotZero(t, testStats["TestSomething"].Start)
	assert.NotZero(t, testStats["TestSomething"].End)
	assert.True(t, testStats["TestSomething"].Passed)

	assert.NotZero(t, testStats["TestPanic"].Start)
	assert.NotZero(t, testStats["TestPanic"].End)
	assert.False(t, testStats["TestPanic"].Passed)
}

// FailfastSuite will test the behavior when running with the failfast flag
// It logs calls in the callOrder slice which we then use to assert the correct calls were made
type FailfastSuite struct {
	*suite.Suite[FailfastSuite, FailfastSuiteGlobalData]
}

var callOrder []string

type FailfastSuiteGlobalData struct{}

func (s *FailfastSuite) call(method string) {
	callOrder = append(callOrder, method)
}

func TestFailfastSuite(t *testing.T) {
	// This test suite is run twice. Once normally and once with the -failfast flag by TestFailfastSuiteFailFastOn
	// If you need to debug it run this test directly with the failfast flag set on/off as you need
	failFast := flag.Lookup("test.failfast").Value.(flag.Getter).Get().(bool)
	ok := testing.RunTests(
		allTestsFilter,
		[]testing.InternalTest{{
			Name: t.Name() + "/FailfastSuite",
			F: func(t *testing.T) {
				suite.Run[FailfastSuite, FailfastSuiteGlobalData](t)
			},
		}},
	)
	assert.False(t, ok)
	var expect []string
	if failFast {
		// Test A Fails and because we are running with failfast Test B never runs and we proceed straight to TearDownSuite
		expect = []string{"SetupSuite", "SetupTest", "Test A Fails", "TearDownTest", "TearDownSuite"}
	} else {
		// Test A Fails and because we are running without failfast we continue and run Test B and then proceed to TearDownSuite
		expect = []string{"SetupSuite", "SetupTest", "Test A Fails", "TearDownTest", "SetupTest", "Test B Passes", "TearDownTest", "TearDownSuite"}
	}
	callOrderAssert(t, expect, callOrder)
}

type tHelper interface {
	Helper()
}

// callOrderAssert is a help with confirms that asserts that expect
// matches one or more times in callOrder. This makes it compatible
// with go test flag -count=X where X > 1.
func callOrderAssert(t *testing.T, expect, callOrder []string) {
	var ti any = t
	if h, ok := ti.(tHelper); ok {
		h.Helper()
	}

	callCount := len(callOrder)
	expectCount := len(expect)
	if callCount > expectCount && callCount%expectCount == 0 {
		// Command line flag -count=X where X > 1.
		for len(callOrder) >= expectCount {
			assert.Equal(t, expect, callOrder[:expectCount])
			callOrder = callOrder[expectCount:]
		}
		return
	}

	assert.Equal(t, expect, callOrder)
}

func TestFailfastSuiteFailFastOn(t *testing.T) {
	// To test this with failfast on (and isolated from other intended test failures in our test suite) we launch it in its own process
	cmd := exec.Command("go", "test", "-v", "-race", "-run", "TestFailfastSuite", "-failfast")
	var out bytes.Buffer
	cmd.Stdout = &out
	t.Log("Running go test -v -race -run TestFailfastSuite -failfast")
	err := cmd.Run()
	t.Log(out.String())
	if err != nil {
		t.Log(err)
		t.Fail()
	}
}

func (s *FailfastSuite) SetupSuite() {
	s.call("SetupSuite")
}

func (s *FailfastSuite) TearDownSuite() {
	s.call("TearDownSuite")
}

func (s *FailfastSuite) SetupTest() {
	s.call("SetupTest")
}

func (s *FailfastSuite) TearDownTest() {
	s.call("TearDownTest")
}

func (s *FailfastSuite) Test_A_Fails() {
	s.call("Test A Fails")
	s.T().Error("Test A meant to fail")
}

func (s *FailfastSuite) Test_B_Passes() {
	s.call("Test B Passes")
	s.Require().True(true)
}
