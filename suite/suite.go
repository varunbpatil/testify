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
	_ "github.com/stretchr/testify/suite" // This is only needed for the `testify.m` flag
)

var (
	// x = exclude
	excludeMethod = flag.String("testify.x", "", "regular expression to exclude tests of the testify suite to run")
)

type Suite[T any, G any] struct {
	*assert.Assertions
	require  *require.Assertions
	testingT *testing.T
	suite    *T // user-defined test suite
	g        *G // global data for the suite
	parent   *T // for subtests, the parent suite instance
}

// T retrieves the current *testing.T context.
func (s *Suite[T, G]) T() *testing.T {
	return s.testingT
}

// G retrieves the global data for the suite.
func (s *Suite[T, G]) G() *G {
	return s.g
}

func (s *Suite[T, G]) Cleanup(f func()) {
	s.T().Cleanup(f)
}

func (s *Suite[T, G]) Failed() bool {
	return s.T().Failed()
}

func (s *Suite[T, G]) Fatal(args ...any) {
	s.T().Fatal(args...)
}

func (s *Suite[T, G]) Fatalf(format string, args ...any) {
	s.T().Fatalf(format, args...)
}

func (s *Suite[T, G]) Helper() {
	s.T().Helper()
}

func (s *Suite[T, G]) Log(args ...any) {
	s.T().Log(args...)
}

func (s *Suite[T, G]) Logf(format string, args ...any) {
	s.T().Logf(format, args...)
}

func (s *Suite[T, G]) Name() string {
	return s.T().Name()
}

func (s *Suite[T, G]) Skip(args ...any) {
	s.T().Skip(args...)
}

func (s *Suite[T, G]) SkipNow() {
	s.T().SkipNow()
}

func (s *Suite[T, G]) Skipf(format string, args ...any) {
	s.T().Skipf(format, args...)
}

func (s *Suite[T, G]) Skipped() bool {
	return s.T().Skipped()
}

func (s *Suite[T, G]) TempDir() string {
	return s.T().TempDir()
}

func (s *Suite[T, G]) Deadline() (deadline time.Time, ok bool) {
	return s.T().Deadline()
}

func (s *Suite[T, G]) Setenv(key, value string) {
	s.T().Setenv(key, value)
}

func (s *Suite[T, G]) Parallel() {
	s.T().Parallel()
}

func (s *Suite[T, G]) Parent() *T {
	return s.parent
}

// setT sets the current *testing.T context.
func (s *Suite[T, G]) setT(testingT *testing.T) {
	if s.T() != nil {
		panic("Suite.testingT already set, can't overwrite")
	}
	s.testingT = testingT
	s.Assertions = assert.New(testingT)
	s.require = require.New(testingT)
}

// setG sets the global data for the suite.
func (s *Suite[T, G]) setG(g *G) {
	if s.G() != nil {
		panic("Suite.g already set, can't overwrite")
	}
	s.g = g
}

// setS sets the suite for the current test.
func (s *Suite[T, G]) setS(suite *T) {
	if s.suite != nil {
		panic("Suite.suite already set, can't overwrite")
	}
	s.suite = suite
}

// setP sets the parent suite for the current test.
func (s *Suite[T, G]) setP(suite *T) {
	if s.parent != nil {
		panic("Suite.parent already set, can't overwrite")
	}
	s.parent = suite
}

// Require returns a require context for suite.
func (s *Suite[T, G]) Require() *require.Assertions {
	if s.T() == nil {
		panic("Suite.testingT not set, can't get Require object")
	}
	return s.require
}

// Assert returns an assert context for suite.
func (s *Suite[T, G]) Assert() *assert.Assertions {
	if s.T() == nil {
		panic("Suite.testingT not set, can't get Assert object")
	}
	return s.Assertions
}

func recoverAndFailOnPanic[T any, G any](s *Suite[T, G]) {
	s.Helper()
	r := recover()
	failOnPanic(s, r)
}

func failOnPanic[T any, G any](s *Suite[T, G], r any) {
	s.Helper()
	if r != nil {
		s.T().Errorf("test panicked: %v\n%s", r, debug.Stack())
		s.T().FailNow()
	}
}

// Run provides suite functionality around golang subtests. It should be
// called in place of t.Run(name, func(t *testing.T)) in test suite code.
// The passed-in func will be executed as a subtest with a fresh instance of t.
// Provides compatibility with go test pkg -run TestSuite/TestName/SubTestName.
func (s *Suite[T, G]) Run(name string, subtest func(suite *T)) bool {
	return s.T().Run(name, func(testingT *testing.T) {
		// Each subtest gets a fresh instance of Suite.
		// The global data is passed through to all new instances.
		newS := &Suite[T, G]{}
		newSuite := new(T)
		newS.setT(testingT)
		newS.setG(s.G())
		newS.setS(newSuite)
		newS.setP(s.suite)

		// This catches panics in the subtest setup and fails the test.
		defer recoverAndFailOnPanic(newS)

		if err := setField(newS.suite, "Suite", newS); err != nil {
			panic("make sure that your test suite embeds `*suite.Suite`")
		}

		// [T.Cleanup] ensures that the teardown method is executed after all the subtests
		// (of this subtest) are done, even in the case of parallel subtests. This cannot be
		// accomplished with a simple defer statement because the deferred function will
		// execute before the subtests even start running (in the case of parallel subtests).
		if tearDownSubTest, ok := any(newSuite).(TearDownSubTest); ok {
			newS.Cleanup(func() {
				defer recoverAndFailOnPanic(newS)
				tearDownSubTest.TearDownSubTest()
			})
		}

		// Setup the subtest.
		if setupSubTest, ok := any(newSuite).(SetupSubTest); ok {
			setupSubTest.SetupSubTest()
		}

		// Call the subtest function with the new instance of the suite.
		// This new instance of suite will have its own testing.T context.
		// as well as per-test data. Global data will be shared.
		subtest(newSuite)
	})
}

// Run runs all of the tests attached to a suite.
func Run[T any, G any](testingT *testing.T) {
	flag.Parse()

	s := &Suite[T, G]{}
	suite := new(T)
	s.setT(testingT)
	s.setG(new(G))
	s.setS(suite)
	s.setP(nil)

	// This catches panics in the test suite setup and fails the test.
	defer recoverAndFailOnPanic(s)

	if err := setField(s.suite, "Suite", s); err != nil {
		panic("make sure that your test suite embeds `*suite.Suite`")
	}

	methodFinder := reflect.TypeOf(suite)
	suiteName := methodFinder.Elem().Name()

	// Iterate over all the methods of the test suite and prepare the list of tests to run.
	var methods []reflect.Method
	for i := 0; i < methodFinder.NumMethod(); i++ {
		method := methodFinder.Method(i)
		ok, err := methodFilter(method.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "testify: invalid regexp: %s\n", err)
			os.Exit(1)
		}
		if ok {
			methods = append(methods, method)
		}
	}

	if len(methods) == 0 {
		testingT.Log("warning: no tests to run")
		return
	}

	// Setup stats.
	var stats *SuiteInformation
	if _, ok := any(suite).(WithStats); ok {
		stats = newSuiteInformation()
	}

	// [T.Cleanup] ensures that the stats handler is called only after all the tests in the
	// suite are done, even in the case of parallel tests. This cannot be accomplished with a
	// simple defer statement because the deferred function will execute before the subtests
	// even start running (in the case of parallel subtests).
	if stats != nil {
		s.Cleanup(func() {
			defer recoverAndFailOnPanic(s)
			stats.End = time.Now()
			if suiteWithStats, ok := any(suite).(WithStats); ok {
				suiteWithStats.HandleStats(suiteName, stats)
			}
		})

		// Start the stats collection.
		stats.Start = time.Now()
	}

	// [T.Cleanup] ensures that the suite teardown method is executed only after all the tests
	// in the suite are done, even in the case of parallel tests. This cannot be accomplished
	// with a simple defer statement because the deferred function will execute before the
	// subtests even start running (in the case of parallel subtests).
	if tearDownAllSuite, ok := any(suite).(TearDownAllSuite); ok {
		s.Cleanup(func() {
			defer recoverAndFailOnPanic(s)
			tearDownAllSuite.TearDownSuite()
		})
	}

	// Setup the suite. The cleanup function is already registered above. So, it will be called
	// even if the setup function panics.
	if setupAllSuite, ok := any(suite).(SetupAllSuite); ok {
		setupAllSuite.SetupSuite()
	}

	// Each method of the test suite is executed as a subtest of the suite.
	// Prepare the list of sub-tests to run.
	tests := []testing.InternalTest{}
	for _, method := range methods {
		method := method
		test := testing.InternalTest{
			Name: method.Name,
			F: func(testingT *testing.T) {
				// Each test gets a fresh instance of [Suite].
				// The global data is passed through to all new instances.
				newS := &Suite[T, G]{}
				newSuite := new(T)
				newS.setT(testingT)
				newS.setG(s.G())
				newS.setS(newSuite)
				newS.setP(s.suite)

				// This catches panics in the test setup and fails the test.
				defer recoverAndFailOnPanic(newS)

				if err := setField(newS.suite, "Suite", newS); err != nil {
					panic("make sure that your test suite embeds `*suite.Suite`")
				}

				// [T.Cleanup] ensures that the stats are updated only after all the
				// sub-tests of this test are done, even in the case of parallel tests.
				if stats != nil {
					newS.Cleanup(func() { stats.end(method.Name, !newS.Failed()) })

					// Start the stats collection.
					stats.start(method.Name)
				}

				// The order of calls are: SetupTest -> BeforeTest -> Test ->
				// AfterTest -> TearDownTest
				//
				// Registering the cleanup methods before their corresponding setup
				// methods ensures that the cleanup methods are always called. This
				// cannot be accomplished with a simple defer statement because the
				// deferred function will execute before the subtests even start
				// running (in the case of parallel subtests).
				if tearDownTestSuite, ok := any(newSuite).(TearDownTestSuite); ok {
					newS.Cleanup(func() {
						defer recoverAndFailOnPanic(newS)
						tearDownTestSuite.TearDownTest()
					})
				}

				if setupTestSuite, ok := any(newSuite).(SetupTestSuite); ok {
					setupTestSuite.SetupTest()
				}

				if afterTestSuite, ok := any(newSuite).(AfterTest); ok {
					newS.Cleanup(func() {
						defer recoverAndFailOnPanic(newS)
						afterTestSuite.AfterTest(suiteName, method.Name)
					})
				}

				if beforeTestSuite, ok := any(newSuite).(BeforeTest); ok {
					beforeTestSuite.BeforeTest(methodFinder.Elem().Name(), method.Name)
				}

				method.Func.Call([]reflect.Value{reflect.ValueOf(newSuite)})
			},
		}

		tests = append(tests, test)
	}

	// Run each test method as a subtest of the suite.
	for _, test := range tests {
		testingT.Run(test.Name, test.F)
	}
}

// Filtering methods according to set regular expression.
func methodFilter(name string) (bool, error) {
	// Exclude methods that don't start with "Test".
	if ok, _ := regexp.MatchString("^Test", name); !ok {
		return false, nil
	}

	// Get the value of the `testify.m` flag.
	testifyM := flag.Lookup("testify.m")

	// If the `testify.m` flag is set, include methods that match the regex.
	if testifyM != nil && testifyM.Value != nil && testifyM.Value.String() != "" {
		return regexp.MatchString(testifyM.Value.String(), name)
	}

	// If the `testify.x` flag is set, exclude methods that match the regex.
	if *excludeMethod != "" {
		match, err := regexp.MatchString(*excludeMethod, name)
		return !match, err
	}

	return true, nil
}

// setField sets the value of a field in a struct.
func setField[T any](input *T, fieldName string, value any) error {
	// Use reflection to get the field by name
	field := reflect.ValueOf(input).Elem().FieldByName(fieldName)
	if !field.IsValid() {
		return fmt.Errorf("field %s does not exist", fieldName)
	}
	if !field.CanSet() {
		return fmt.Errorf("field %s cannot be set", fieldName)
	}

	// Check if the value's type is compatible with the field's type
	valueRef := reflect.ValueOf(value)
	if !valueRef.Type().AssignableTo(field.Type()) {
		return fmt.Errorf(
			"value of type %s is not assignable to field %s of type %s",
			valueRef.Type(), fieldName, field.Type(),
		)
	}

	field.Set(valueRef)
	return nil
}
