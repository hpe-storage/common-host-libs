// Copyright 2026 Hewlett Packard Enterprise Development LP

package leaderlease

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	kubetesting "k8s.io/client-go/testing"

	"k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "hpe-storage"

func TestNewRequiresLeaseNameAndNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	_, err := New(clientset, "pod-a", Config{Namespace: testNamespace})
	assert.Error(t, err, "LeaseName is required")

	_, err = New(clientset, "pod-a", Config{LeaseName: "test-lease"})
	assert.Error(t, err, "Namespace is required")
}

func TestStartAcquiresLeaseAndInvokesOnStartedLeading(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	// Buffered channel, not a plain variable: OnStartedLeading runs on client-go's own goroutine
	// (started via `go` in LeaderElector.Run), so a plain write here would race against this
	// goroutine's read below - IsLeader() becoming true only synchronizes the atomic itself, not
	// gotCtx, which is written after it.
	gotCtx := make(chan context.Context, 1)
	e, err := New(clientset, "pod-a", Config{
		LeaseName:     "test-lease",
		Namespace:     testNamespace,
		LeaseDuration: 100 * time.Millisecond,
		RenewDeadline: 60 * time.Millisecond,
		RetryPeriod:   20 * time.Millisecond,
		OnStartedLeading: func(ctx context.Context) {
			gotCtx <- ctx
		},
	})
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	assert.NoError(t, e.Start(ctx))

	assert.Eventually(t, e.IsLeader, time.Second, 5*time.Millisecond)
	select {
	case got := <-gotCtx:
		assert.NotNil(t, got)
	case <-time.After(time.Second):
		t.Fatal("OnStartedLeading was never invoked")
	}
}

func TestExternalContextCancellationIsTreatedAsGracefulNotUnexpectedLoss(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	exitCalls := 0
	e, err := New(clientset, "pod-a", Config{
		LeaseName:            "test-lease",
		Namespace:            testNamespace,
		LeaseDuration:        100 * time.Millisecond,
		RenewDeadline:        60 * time.Millisecond,
		RetryPeriod:          20 * time.Millisecond,
		ExitOnUnexpectedLoss: true, // the exact policy under which a misclassification would exit
		Exit:                 func(int) { exitCalls++ },
	})
	assert.NoError(t, err)

	// Cancel the context passed to Start directly, bypassing Shutdown entirely - simulates a
	// future caller that doesn't know Shutdown is the only "safe" way to stop.
	ctx, cancel := context.WithCancel(context.Background())
	assert.NoError(t, e.Start(ctx))
	assert.Eventually(t, e.IsLeader, time.Second, 5*time.Millisecond)

	cancel()

	assert.Eventually(t, func() bool { return !e.IsLeader() }, time.Second, 5*time.Millisecond)
	time.Sleep(100 * time.Millisecond) // give a misclassifying implementation a chance to call Exit
	assert.Equal(t, 0, exitCalls, "direct ctx cancellation must be treated as graceful, never as an unexpected loss")
}

func TestShutdownStopsCleanlyAndDoesNotExit(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	exitCalls := 0
	e, err := New(clientset, "pod-a", Config{
		LeaseName:            "test-lease",
		Namespace:            testNamespace,
		LeaseDuration:        100 * time.Millisecond,
		RenewDeadline:        60 * time.Millisecond,
		RetryPeriod:          20 * time.Millisecond,
		ExitOnUnexpectedLoss: true, // Shutdown must still not exit - that's only for unexpected loss
		Exit:                 func(int) { exitCalls++ },
	})
	assert.NoError(t, err)

	ctx := context.Background()
	assert.NoError(t, e.Start(ctx))
	assert.Eventually(t, e.IsLeader, time.Second, 5*time.Millisecond)

	drainCalled := false
	e.Shutdown(func() { drainCalled = true })

	assert.True(t, drainCalled, "drain must be called before releasing the lease")
	assert.False(t, e.IsLeader())
	assert.Equal(t, 0, exitCalls, "graceful shutdown must never exit the process")
}

func TestExitOnUnexpectedLossExitsOnceAndDoesNotReContest(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	var updateCalls atomic.Int32
	clientset.PrependReactor("update", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		if updateCalls.Add(1) <= 1 {
			return false, nil, nil // let the first renewal succeed normally
		}
		// Force every subsequent renewal to fail until RenewDeadline is exhausted, simulating an
		// unexpected loss (e.g. a transient apiserver blip outlasting the renew deadline).
		return true, nil, apierrors.NewServerTimeout(coordinationv1.Resource("leases"), "update", 0)
	})

	var startedCount, exitCalls atomic.Int32
	e, err := New(clientset, "pod-a", Config{
		LeaseName:            "test-lease",
		Namespace:            testNamespace,
		LeaseDuration:        100 * time.Millisecond,
		RenewDeadline:        60 * time.Millisecond,
		RetryPeriod:          20 * time.Millisecond,
		OnStartedLeading:     func(context.Context) { startedCount.Add(1) },
		ExitOnUnexpectedLoss: true,
		Exit:                 func(int) { exitCalls.Add(1) },
	})
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	assert.NoError(t, e.Start(ctx))

	assert.Eventually(t, func() bool { return exitCalls.Load() == 1 }, 1500*time.Millisecond, 10*time.Millisecond,
		"an unexpected loss with ExitOnUnexpectedLoss must call Exit exactly once")
	time.Sleep(200 * time.Millisecond) // give a wrongly-looping implementation a chance to re-acquire
	assert.Equal(t, int32(1), startedCount.Load(), "must not re-contest after exiting")
}

func TestReContestsAfterUnexpectedLossWhenNotExitingOnLoss(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	var updateCalls atomic.Int32
	clientset.PrependReactor("update", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		// Fail 4 consecutive renewals belonging to the first acquisition - matching the number of
		// retries (RenewDeadline/RetryPeriod = 60ms/20ms = 3, plus the initial attempt) actually
		// observed before client-go gives up and Run returns - then let every later call (the
		// re-contest's own create/renewals) succeed normally.
		call := updateCalls.Add(1)
		if call >= 2 && call <= 5 {
			return true, nil, apierrors.NewServerTimeout(coordinationv1.Resource("leases"), "update", 0)
		}
		return false, nil, nil
	})

	var startedCount atomic.Int32
	e, err := New(clientset, "pod-a", Config{
		LeaseName:            "test-lease",
		Namespace:            testNamespace,
		LeaseDuration:        100 * time.Millisecond,
		RenewDeadline:        60 * time.Millisecond,
		RetryPeriod:          20 * time.Millisecond,
		OnStartedLeading:     func(context.Context) { startedCount.Add(1) },
		ExitOnUnexpectedLoss: false,
	})
	assert.NoError(t, err)

	// The re-contest backoff is a jittered wait.Jitter(2s, 1.0) (i.e. 2-4s), so allow enough
	// headroom for that plus the initial acquire/renewal-failure window.
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	assert.NoError(t, e.Start(ctx))

	assert.Eventually(t, func() bool { return startedCount.Load() >= 2 }, 6*time.Second, 20*time.Millisecond,
		"expected the elector to re-contest and acquire leadership again after an unexpected loss (H1)")
}

func TestOnStoppedLeadingReportsWasLeaderCorrectly(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	var gotWasLeader []bool
	e, err := New(clientset, "pod-a", Config{
		LeaseName:        "test-lease",
		Namespace:        testNamespace,
		LeaseDuration:    100 * time.Millisecond,
		RenewDeadline:    60 * time.Millisecond,
		RetryPeriod:      20 * time.Millisecond,
		OnStoppedLeading: func(wasLeader bool) { gotWasLeader = append(gotWasLeader, wasLeader) },
	})
	assert.NoError(t, err)

	ctx := context.Background()
	assert.NoError(t, e.Start(ctx))
	assert.Eventually(t, e.IsLeader, time.Second, 5*time.Millisecond)

	e.Shutdown(func() {})

	assert.Equal(t, []bool{true}, gotWasLeader)
}
