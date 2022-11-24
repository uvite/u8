package tests

import (
	"context"
	"io/ioutil"
	"net/url"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/uvite/u8/cmd/integration_tests/testmodules/events"
	"github.com/uvite/u8/core/local"
	"github.com/uvite/u8/js"
	"github.com/uvite/u8/js/modules"
	"github.com/uvite/u8/lib"
	"github.com/uvite/u8/lib/executor"
	"github.com/uvite/u8/lib/testutils"
	"github.com/uvite/u8/lib/types"
	"github.com/uvite/u8/loader"
	"github.com/uvite/u8/metrics"
	"gopkg.in/guregu/null.v3"
)

func eventLoopTest(t *testing.T, script []byte, testHandle func(context.Context, lib.Runner, error, *testutils.SimpleLogrusHook)) {
	logger := logrus.New()
	logger.SetOutput(ioutil.Discard)
	logHook := &testutils.SimpleLogrusHook{HookedLevels: []logrus.Level{logrus.InfoLevel, logrus.WarnLevel, logrus.ErrorLevel}}
	logger.AddHook(logHook)

	registry := metrics.NewRegistry()
	piState := &lib.TestPreInitState{
		Logger:         logger,
		Registry:       registry,
		BuiltinMetrics: metrics.RegisterBuiltinMetrics(registry),
	}

	script = []byte("import {setTimeout} from 'k6/x/events';\n" + string(script))
	runner, err := js.New(piState, &loader.SourceData{URL: &url.URL{Path: "/script.js"}, Data: script}, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	newOpts, err := executor.DeriveScenariosFromShortcuts(lib.Options{
		MetricSamplesBufferSize: null.NewInt(200, false),
		TeardownTimeout:         types.NullDurationFrom(time.Second),
		SetupTimeout:            types.NullDurationFrom(time.Second),
	}.Apply(runner.GetOptions()), nil)
	require.NoError(t, err)
	require.Empty(t, newOpts.Validate())
	require.NoError(t, runner.SetOptions(newOpts))

	testState := &lib.TestRunState{
		TestPreInitState: piState,
		Options:          newOpts,
		Runner:           runner,
		RunTags:          piState.Registry.RootTagSet().WithTagsFromMap(newOpts.RunTags),
	}

	execScheduler, err := local.NewExecutionScheduler(testState)
	require.NoError(t, err)

	samples := make(chan metrics.SampleContainer, newOpts.MetricSamplesBufferSize.Int64)
	go func() {
		for {
			select {
			case <-samples:
			case <-ctx.Done():
				return
			}
		}
	}()

	require.NoError(t, execScheduler.Init(ctx, samples))

	errCh := make(chan error, 1)
	go func() { errCh <- execScheduler.Run(ctx, ctx, samples) }()

	select {
	case err := <-errCh:
		testHandle(ctx, runner, err, logHook)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out")
	}
}

func init() {
	modules.Register("k6/x/events", events.New())
}

func TestEventLoop(t *testing.T) {
	t.Parallel()
	script := []byte(`
		setTimeout(()=> {console.log("initcontext setTimeout")}, 200)
		console.log("initcontext");
		export default function() {
			setTimeout(()=> {console.log("default setTimeout")}, 200)
			console.log("default");
		};
		export function setup() {
			setTimeout(()=> {console.log("setup setTimeout")}, 200)
			console.log("setup");
		};
		export function teardown() {
			setTimeout(()=> {console.log("teardown setTimeout")}, 200)
			console.log("teardown");
		};
		export function handleSummary() {
			setTimeout(()=> {console.log("handleSummary setTimeout")}, 200)
			console.log("handleSummary");
		};
`)
	eventLoopTest(t, script, func(ctx context.Context, runner lib.Runner, err error, logHook *testutils.SimpleLogrusHook) {
		require.NoError(t, err)
		_, err = runner.HandleSummary(ctx, &lib.Summary{RootGroup: &lib.Group{}})
		require.NoError(t, err)
		entries := logHook.Drain()
		msgs := make([]string, len(entries))
		for i, entry := range entries {
			msgs[i] = entry.Message
		}
		require.Equal(t, []string{
			"initcontext", // first initialization
			"initcontext setTimeout",
			"initcontext", // for vu
			"initcontext setTimeout",
			"initcontext", // for setup
			"initcontext setTimeout",
			"setup", // setup
			"setup setTimeout",
			"default", // one iteration
			"default setTimeout",
			"initcontext", // for teardown
			"initcontext setTimeout",
			"teardown", // teardown
			"teardown setTimeout",
			"initcontext", // for handleSummary
			"initcontext setTimeout",
			"handleSummary", // handleSummary
			"handleSummary setTimeout",
		}, msgs)
	})
}

func TestEventLoopCrossScenario(t *testing.T) {
	t.Parallel()
	script := []byte(`
import exec from "k6/execution"
export const options = {
        scenarios: {
                "first":{
                        executor: "shared-iterations",
                        maxDuration: "1s",
                        iterations: 1,
                        vus: 1,
                        gracefulStop:"1s",
                },
                "second": {
                        executor: "shared-iterations",
                        maxDuration: "1s",
                        iterations: 1,
                        vus: 1,
                        startTime: "3s",
                }
        }
}
export default function() {
	let i = exec.scenario.name
	setTimeout(()=> {console.log(i)}, 3000)
}
`)

	eventLoopTest(t, script, func(_ context.Context, _ lib.Runner, err error, logHook *testutils.SimpleLogrusHook) {
		require.NoError(t, err)
		entries := logHook.Drain()
		msgs := make([]string, len(entries))
		for i, entry := range entries {
			msgs[i] = entry.Message
		}
		require.Equal(t, []string{
			"setTimeout 1 was stopped because the VU iteration was interrupted",
			"second",
		}, msgs)
	})
}

func TestEventLoopDoesntCrossIterations(t *testing.T) {
	t.Parallel()
	script := []byte(`
import { sleep } from "k6"
export const options = {
  iterations: 2,
  vus: 1,
}

export default function() {
  let i = __ITER;
	setTimeout(()=> { console.log(i) }, 1000)
  if (__ITER == 0) {
    throw "just error"
  } else {
    sleep(1)
  }
}
`)

	eventLoopTest(t, script, func(_ context.Context, _ lib.Runner, err error, logHook *testutils.SimpleLogrusHook) {
		require.NoError(t, err)
		entries := logHook.Drain()
		msgs := make([]string, len(entries))
		for i, entry := range entries {
			msgs[i] = entry.Message
		}
		require.Equal(t, []string{
			"setTimeout 1 was stopped because the VU iteration was interrupted",
			"just error\n\tat /script.js:13:4(15)\n\tat native\n", "1",
		}, msgs)
	})
}