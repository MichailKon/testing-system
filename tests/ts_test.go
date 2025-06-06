package tests

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"os"
	"sync"
	"testing"
	"time"
)

func runSanbodxTests(t *testing.T, test func(*testing.T, string)) {
	t.Run("Simple sandbox", func(t *testing.T) {
		test(t, "simple")
	})

	t.Run("Isolate sandbox", func(t *testing.T) {
		_, err := os.Stat("/usr/local/bin/isolate")
		if os.IsNotExist(err) {
			t.Skip("isolate sanbox not installed, skipping all testing system tests with isolate sandbox")
		} else if err != nil {
			t.Fatal(err)
		}
		test(t, "isolate")
	})
}

func TestTSInit(t *testing.T) {
	runSanbodxTests(t, testTSInit)
}

func testTSInit(t *testing.T, sandbox string) {
	h := initTS(t, sandbox)
	go h.start()
	time.Sleep(10 * time.Second)
	h.stop()
}

func TestTSPanic(t *testing.T) {
	runSanbodxTests(t, testTSPanic)
}

func testTSPanic(t *testing.T, sandbox string) {
	h := initTS(t, sandbox)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		require.Panics(t, h.start)
		wg.Done()
	}()
	h.ts.Go(func() {
		// Wait until testing system is set up
		time.Sleep(10 * time.Millisecond)
		panic("PANIC!!!")
	})
	wg.Wait()
}

func TestSingleSubmit(t *testing.T) {
	runSanbodxTests(t, testSingleSubmit)
}

func testSingleSubmit(t *testing.T, sandbox string) {
	h := initTS(t, sandbox)
	go h.start()
	// Wait until TS is ready
	time.Sleep(10 * time.Millisecond)

	h.newSubmit(1)
	h.waitSubmits()

	h.stop()
}

func TestMultiSubmit(t *testing.T) {
	runSanbodxTests(t, testMultiSubmit)
}

func testMultiSubmit(t *testing.T, sandbox string) {
	h := initTS(t, sandbox)
	go h.start()
	// Wait until TS is ready
	time.Sleep(10 * time.Millisecond)

	h.newSubmit(1)
	h.newSubmit(2)
	h.newSubmit(3)
	h.newSubmit(4)
	h.newSubmit(5)
	if sandbox == "isolate" {
		h.newSubmit(6) // We can not detect ML with simple sandbox
	}
	h.newSubmit(7)

	h.waitSubmits()
	h.stop()
}

func TestSkipJobs(t *testing.T) {
	runSanbodxTests(t, testSkipJobs)
}

func testSkipJobs(t *testing.T, sandbox string) {
	h := initTS(t, sandbox)
	go h.start()
	time.Sleep(10 * time.Millisecond)
	h.newSubmit(8)
	h.waitSubmits()
	metrics := make(chan prometheus.Metric, 100)

	h.ts.Metrics.InvokerSkippedJobs.Collect(metrics)
	close(metrics)
	var skippedJobs float64
	for singleMetric := range metrics {
		var metric dto.Metric
		require.NoError(t, singleMetric.Write(&metric))
		skippedJobs += metric.Counter.GetValue()
	}
	require.Greater(t, skippedJobs, float64(0))
	h.stop()
}

func TestLargeQueue(t *testing.T) {
	runSanbodxTests(t, testLargeQueue)
}

func testLargeQueue(t *testing.T, sandbox string) {
	h := initTS(t, sandbox)
	go h.start()
	time.Sleep(10 * time.Millisecond)
	for _ = range 100 {
		h.newSubmit(1)
	}
	h.waitSubmits()
	h.stop()
}
