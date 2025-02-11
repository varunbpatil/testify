package suite

// SetupAllSuite has a SetupSuite method, which will run before the
// tests in the suite are run.
type SetupAllSuite[D any] interface {
	SetupSuite(t *T[D])
}

// SetupTestSuite has a SetupTest method, which will run before each
// test in the suite.
type SetupTestSuite[D any] interface {
	SetupTest(t *T[D])
}

// TearDownAllSuite has a TearDownSuite method, which will run after
// all the tests in the suite have been run.
type TearDownAllSuite[D any] interface {
	TearDownSuite(t *T[D])
}

// TearDownTestSuite has a TearDownTest method, which will run after
// each test in the suite.
type TearDownTestSuite[D any] interface {
	TearDownTest(t *T[D])
}

// BeforeTest has a function to be executed right before the test
// starts and receives the suite and test names as input
type BeforeTest[D any] interface {
	BeforeTest(t *T[D], suiteName, testName string)
}

// AfterTest has a function to be executed right after the test
// finishes and receives the suite and test names as input
type AfterTest[D any] interface {
	AfterTest(t *T[D], suiteName, testName string)
}

// WithStats implements HandleStats, a function that will be executed
// when a test suite is finished. The stats contain information about
// the execution of that suite and its tests.
type WithStats[D any] interface {
	HandleStats(t *T[D], suiteName string, stats *SuiteInformation)
}

// SetupSubTest has a SetupSubTest method, which will run before each
// subtest in the suite.
type SetupSubTest[D any] interface {
	SetupSubTest(t *T[D])
}

// TearDownSubTest has a TearDownSubTest method, which will run after
// each subtest in the suite have been run.
type TearDownSubTest[D any] interface {
	TearDownSubTest(t *T[D])
}
