package suite

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var matchMethod = flag.String("testify-parallel.m", "", "regular expression to select tests of the testify suite to run")

type T[D any] struct {
	*assert.Assertions
	require  *require.Assertions
	testingT *testing.T
	suite    any
	testData *D
}

// T retrieves the current *testing.T context.
func (t *T[D]) T() *testing.T {
	return t.testingT
}

// D retrieves the data for the current test.
func (t *T[D]) D() *D {
	return t.testData
}

func (t *T[D]) Cleanup(f func()) {
	t.testingT.Cleanup(f)
}

func (t *T[D]) Error(args ...interface{}) {
	t.testingT.Error(args...)
}

func (t *T[D]) Errorf(format string, args ...interface{}) {
	t.testingT.Errorf(format, args...)
}

func (t *T[D]) Fail() {
	t.testingT.Fail()
}

func (t *T[D]) FailNow() {
	t.testingT.FailNow()
}

func (t *T[D]) Failed() bool {
	return t.testingT.Failed()
}

func (t *T[D]) Fatal(args ...interface{}) {
	t.testingT.Fatal(args...)
}

func (t *T[D]) Fatalf(format string, args ...interface{}) {
	t.testingT.Fatalf(format, args...)
}

func (t *T[D]) Helper() {
	t.testingT.Helper()
}

func (t *T[D]) Log(args ...interface{}) {
	t.testingT.Log(args...)
}

func (t *T[D]) Logf(format string, args ...interface{}) {
	t.testingT.Logf(format, args...)
}

func (t *T[D]) Name() string {
	return t.testingT.Name()
}

func (t *T[D]) Skip(args ...interface{}) {
	t.testingT.Skip(args...)
}

func (t *T[D]) SkipNow() {
	t.testingT.SkipNow()
}

func (t *T[D]) Skipf(format string, args ...interface{}) {
	t.testingT.Skipf(format, args...)
}

func (t *T[D]) Skipped() bool {
	return t.testingT.Skipped()
}

func (t *T[D]) TempDir() string {
	return t.testingT.TempDir()
}

func (t *T[D]) Deadline() (deadline time.Time, ok bool) {
	return t.testingT.Deadline()
}

func (t *T[D]) Setenv(key, value string) {
	t.testingT.Setenv(key, value)
}

func (t *T[D]) Parallel() {
	t.testingT.Parallel()
}

// setT sets the current *testing.T context.
func (t *T[D]) setT(testingT *testing.T) {
	if t.testingT != nil {
		panic("T.testingT already set, can't overwrite")
	}
	t.testingT = testingT
	t.Assertions = assert.New(testingT)
	t.require = require.New(testingT)
}

// setD sets the data for the current test.
func (t *T[D]) setD(testData *D) {
	if t.testData != nil {
		panic("T.testData already set, can't overwrite")
	}
	t.testData = testData
}

// setS sets the suite for the current test.
func (t *T[D]) setS(suite any) {
	if t.suite != nil {
		panic("T.suite already set, can't overwrite")
	}
	t.suite = suite
}

// Require returns a require context for suite.
func (t *T[D]) Require() *require.Assertions {
	if t.testingT == nil {
		panic("T.testingT not set, can't get Require object")
	}
	return t.require
}

// Assert returns an assert context for suite.
func (t *T[D]) Assert() *assert.Assertions {
	if t.testingT == nil {
		panic("T.testingT not set, can't get Assert object")
	}
	return t.Assertions
}

func recoverAndFailOnPanic[D any](t *T[D]) {
	t.Helper()
	r := recover()
	failOnPanic(t, r)
}

func failOnPanic[D any](t *T[D], r interface{}) {
	t.Helper()
	if r != nil {
		t.Errorf("test panicked: %v\n%s", r, debug.Stack())
		t.FailNow()
	}
}

// Run provides suite functionality around golang subtests. It should be
// called in place of t.Run(name, func(t *testing.T)) in test suite code.
// The passed-in func will be executed as a subtest with a fresh instance of t.
// Provides compatibility with go test pkg -run TestSuite/TestName/SubTestName.
func (t *T[D]) Run(name string, subtest func(t *T[D])) bool {
	return t.testingT.Run(name, func(testingT *testing.T) {
		// Each subtest gets a fresh instance of T.
		newT := &T[D]{}
		newT.setT(testingT)
		newT.setS(t.suite)

		// Each subtest gets a fresh instance of per-test data.
		// The initialization for this data is defined by the
		// caller by implementing [SetupSubTest] in the suite.
		var testData D
		newT.setD(&testData)

		defer recoverAndFailOnPanic(newT)

		if setupSubTest, ok := newT.suite.(SetupSubTest[D]); ok {
			setupSubTest.SetupSubTest(newT)
		}

		if tearDownSubTest, ok := newT.suite.(TearDownSubTest[D]); ok {
			// [T.Cleanup] ensures that the teardown method is executed after all
			// the subtests are done, even when parallel subtests are used.
			newT.Cleanup(func() { tearDownSubTest.TearDownSubTest(newT) })
		}

		subtest(newT)
	})
}

// Run takes a testing suite and runs all of the tests attached to it.
func Run[D any](testingT *testing.T, suite any) {
	t := &T[D]{}
	t.setT(testingT)
	t.setS(suite)

	var testData D
	t.setD(&testData)

	defer recoverAndFailOnPanic(t)

	var stats *SuiteInformation
	if _, ok := suite.(WithStats[D]); ok {
		stats = newSuiteInformation()
	}

	tests := []testing.InternalTest{}
	methodFinder := reflect.TypeOf(suite)
	suiteName := methodFinder.Elem().Name()
	suiteSetupDone := false

	for i := 0; i < methodFinder.NumMethod(); i++ {
		method := methodFinder.Method(i)

		ok, err := methodFilter(method.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "testify: invalid regexp for -m: %s\n", err)
			os.Exit(1)
		}
		if !ok {
			continue
		}

		if !suiteSetupDone {
			if stats != nil {
				stats.Start = time.Now()
			}

			if setupAllSuite, ok := suite.(SetupAllSuite[D]); ok {
				setupAllSuite.SetupSuite(t)
			}

			suiteSetupDone = true
		}

		test := testing.InternalTest{
			Name: method.Name,
			F: func(testingT *testing.T) {
				// Each test gets a fresh instance of T.
				t := &T[D]{}
				t.setT(testingT)
				t.setS(suite)

				// Each test gets a fresh instance of per-test data.
				// The initialization for this data is defined by the
				// caller by implementing [SetupTestSuite] in the suite.
				var testData D
				t.setD(&testData)

				defer recoverAndFailOnPanic(t)
				defer func() {
					t.Helper()
					r := recover()

					if stats != nil {
						passed := !t.Failed() && r == nil
						stats.end(method.Name, passed)
					}

					failOnPanic(t, r)
				}()

				// The order of calls are:
				// SetupTest -> BeforeTest -> Test -> AfterTest -> TearDownTest
				//
				// Note the use of [T.Cleanup] which ensures that the teardown
				// methods are executed only after all the tests are done, even
				// in the case of parallel tests. Methods registered with [T.Cleanup]
				// are executed in the last added, first called order.
				if setupTestSuite, ok := suite.(SetupTestSuite[D]); ok {
					setupTestSuite.SetupTest(t)
				}

				if beforeTestSuite, ok := suite.(BeforeTest[D]); ok {
					beforeTestSuite.BeforeTest(t, methodFinder.Elem().Name(), method.Name)
				}

				if tearDownTestSuite, ok := suite.(TearDownTestSuite[D]); ok {
					t.Cleanup(func() { tearDownTestSuite.TearDownTest(t) })
				}

				if afterTestSuite, ok := suite.(AfterTest[D]); ok {
					t.Cleanup(func() { afterTestSuite.AfterTest(t, suiteName, method.Name) })
				}

				if stats != nil {
					stats.start(method.Name)
				}

				method.Func.Call([]reflect.Value{reflect.ValueOf(suite), reflect.ValueOf(t)})
			},
		}

		tests = append(tests, test)
	}

	if len(tests) == 0 {
		testingT.Log("warning: no tests to run")
		return
	}

	defer func() {
		t.Helper()
		r := recover()

		if stats != nil {
			stats.End = time.Now()

			if suiteWithStats, measureStats := suite.(WithStats[D]); measureStats {
				suiteWithStats.HandleStats(t, suiteName, stats)
			}
		}

		failOnPanic(t, r)
	}()

	// [T.Cleanup] ensures that the suite teardown method is executed
	// only after all the tests are done, even in the case of parallel tests.
	if tearDownAllSuite, ok := suite.(TearDownAllSuite[D]); ok {
		t.Cleanup(func() { tearDownAllSuite.TearDownSuite(t) })
	}

	// Run each test method as a subtest of the suite.
	for _, test := range tests {
		testingT.Run(test.Name, test.F)
	}
}

// Filtering method according to set regular expression
// specified command-line argument -m
func methodFilter(name string) (bool, error) {
	if ok, _ := regexp.MatchString("^Test", name); !ok {
		return false, nil
	}
	return regexp.MatchString(*matchMethod, name)
}
