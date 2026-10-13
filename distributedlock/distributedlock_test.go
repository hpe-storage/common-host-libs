// Copyright 2026 Hewlett Packard Enterprise Development LP

package distributedlock

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	coordinationv1 "k8s.io/api/coordination/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "k8s.io/client-go/kubernetes"
	coordinationv1client "k8s.io/client-go/kubernetes/typed/coordination/v1"
	kubetesting "k8s.io/client-go/testing"

	"k8s.io/client-go/kubernetes/fake"
)

const testDedupNamespace = "hpe-storage"

// noMaxHold is used by tests that aren't exercising the max-hold-duration ceiling.
const noMaxHold = time.Hour

func TestTryAcquireRelease(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	d := New(testDedupNamespace, "pod-a", clientset, time.Minute, noMaxHold)
	key := "CreateVolumeGroup:vg-1"

	assert.NoError(t, d.TryAcquire(context.Background(), key))

	// Second acquire on the same key, before release, must be reported as a duplicate.
	err := d.TryAcquire(context.Background(), key)
	assert.ErrorIs(t, err, ErrDuplicateInFlight)

	assert.NoError(t, d.Release(context.Background(), key))

	// After release, the key can be acquired again.
	assert.NoError(t, d.TryAcquire(context.Background(), key))
}

func TestTryAcquireStealsStaleLease(t *testing.T) {
	key := "UpdateVolumeMembership:vg-1-vol-1"
	name := leaseNameForKey(key)
	old := metav1.NewTime(time.Now().Add(-time.Hour))
	holder := "pod-crashed"
	ttl := int32(60)
	staleLease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testDedupNamespace,
			Labels:    map[string]string{dedupLeaseLabelKey: dedupLeaseLabelValue},
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       &holder,
			AcquireTime:          &metav1.MicroTime{Time: old.Time},
			RenewTime:            &metav1.MicroTime{Time: old.Time},
			LeaseDurationSeconds: &ttl,
		},
	}
	clientset := fake.NewSimpleClientset(staleLease)
	d := New(testDedupNamespace, "pod-b", clientset, 60*time.Second, noMaxHold)

	assert.NoError(t, d.TryAcquire(context.Background(), key), "a stale lease should be stolen, not reported as a duplicate")
}

func TestTryAcquireConflictOnRace(t *testing.T) {
	key := "CreateVolumeGroup:vg-1"
	name := leaseNameForKey(key)
	old := metav1.NewTime(time.Now().Add(-time.Hour))
	holder := "pod-crashed"
	ttl := int32(60)
	staleLease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testDedupNamespace,
			Labels:    map[string]string{dedupLeaseLabelKey: dedupLeaseLabelValue},
		},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity:       &holder,
			AcquireTime:          &metav1.MicroTime{Time: old.Time},
			RenewTime:            &metav1.MicroTime{Time: old.Time},
			LeaseDurationSeconds: &ttl,
		},
	}
	clientset := fake.NewSimpleClientset(staleLease)
	// Force the next Update() on this Lease to fail with Conflict, simulating a racing replica
	// winning the steal first.
	clientset.PrependReactor("update", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewConflict(coordinationv1.Resource("leases"), name, assert.AnError)
	})
	d := New(testDedupNamespace, "pod-b", clientset, 60*time.Second, noMaxHold)

	err := d.TryAcquire(context.Background(), key)
	assert.ErrorIs(t, err, ErrDuplicateInFlight, "a Conflict on steal must be treated as a duplicate, not an internal error")
}

func TestTryAcquireRetriesOnTransientCreateError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	var createCalls atomic.Int32
	clientset.PrependReactor("create", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		if createCalls.Add(1) == 1 {
			return true, nil, apierrors.NewServiceUnavailable("apiserver hiccup")
		}
		return false, nil, nil // second attempt: let the fake client's default reactor create it
	})
	d := New(testDedupNamespace, "pod-a", clientset, time.Minute, noMaxHold)

	assert.NoError(t, d.TryAcquire(context.Background(), "CreateVolumeGroup:vg-1"),
		"a transient Create failure must be retried and ultimately succeed, not fail the caller's RPC")
	assert.Equal(t, int32(2), createCalls.Load())
}

func TestTryAcquireDoesNotRetryPermanentError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	var createCalls atomic.Int32
	clientset.PrependReactor("create", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		createCalls.Add(1)
		return true, nil, apierrors.NewForbidden(coordinationv1.Resource("leases"), "hpe-csi-dedup-x", assert.AnError)
	})
	d := New(testDedupNamespace, "pod-a", clientset, time.Minute, noMaxHold)

	err := d.TryAcquire(context.Background(), "CreateVolumeGroup:vg-1")
	assert.Error(t, err)
	assert.True(t, apierrors.IsForbidden(errors.Unwrap(err)) || apierrors.IsForbidden(err), "the original Forbidden error must be surfaced")
	assert.Equal(t, int32(1), createCalls.Load(), "a permanent error (e.g. missing RBAC) must not be retried")
}

func TestTryAcquireDoesNotRetryDuplicateInFlight(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	var createCalls atomic.Int32
	clientset.PrependReactor("create", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		createCalls.Add(1)
		return false, nil, nil
	})
	d := New(testDedupNamespace, "pod-a", clientset, time.Minute, noMaxHold)
	key := "CreateVolumeGroup:vg-1"
	assert.NoError(t, d.TryAcquire(context.Background(), key))
	createCalls.Store(0)

	err := d.TryAcquire(context.Background(), key)
	assert.ErrorIs(t, err, ErrDuplicateInFlight)
	assert.Equal(t, int32(1), createCalls.Load(), "a genuine in-flight duplicate must not be retried")
}

func TestReaperDeletesOnlyStaleLabeledLeases(t *testing.T) {
	old := metav1.NewTime(time.Now().Add(-time.Hour))
	recent := metav1.NewTime(time.Now())
	holder := "pod-x"
	ttl := int32(60)

	staleLabeled := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: "hpe-csi-dedup-stale", Namespace: testDedupNamespace, Labels: map[string]string{dedupLeaseLabelKey: dedupLeaseLabelValue}},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder, RenewTime: &metav1.MicroTime{Time: old.Time}, LeaseDurationSeconds: &ttl},
	}
	freshLabeled := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: "hpe-csi-dedup-fresh", Namespace: testDedupNamespace, Labels: map[string]string{dedupLeaseLabelKey: dedupLeaseLabelValue}},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder, RenewTime: &metav1.MicroTime{Time: recent.Time}, LeaseDurationSeconds: &ttl},
	}
	unrelated := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: "some-other-lease", Namespace: testDedupNamespace},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder, RenewTime: &metav1.MicroTime{Time: old.Time}, LeaseDurationSeconds: &ttl},
	}

	clientset := fake.NewSimpleClientset(staleLabeled, freshLabeled, unrelated)
	d := New(testDedupNamespace, "pod-x", clientset, 60*time.Second, noMaxHold)

	d.reapOnce(context.Background())

	_, err := clientset.CoordinationV1().Leases(testDedupNamespace).Get(context.Background(), "hpe-csi-dedup-stale", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "stale labeled lease should have been reaped")

	_, err = clientset.CoordinationV1().Leases(testDedupNamespace).Get(context.Background(), "hpe-csi-dedup-fresh", metav1.GetOptions{})
	assert.NoError(t, err, "fresh labeled lease must not be reaped")

	_, err = clientset.CoordinationV1().Leases(testDedupNamespace).Get(context.Background(), "some-other-lease", metav1.GetOptions{})
	assert.NoError(t, err, "unrelated unlabeled lease must never be touched by the reaper")
}

func TestReapOnceUsesPreconditionsFromListSnapshot(t *testing.T) {
	old := metav1.NewTime(time.Now().Add(-time.Hour))
	holder := "pod-crashed"
	ttl := int32(60)
	staleLease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: "hpe-csi-dedup-stale", Namespace: testDedupNamespace, Labels: map[string]string{dedupLeaseLabelKey: dedupLeaseLabelValue}},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder, RenewTime: &metav1.MicroTime{Time: old.Time}, LeaseDurationSeconds: &ttl},
	}
	clientset := fake.NewSimpleClientset(staleLease)
	listed, err := clientset.CoordinationV1().Leases(testDedupNamespace).Get(context.Background(), "hpe-csi-dedup-stale", metav1.GetOptions{})
	assert.NoError(t, err)

	var gotPreconditions *metav1.Preconditions
	clientset.PrependReactor("delete", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		gotPreconditions = action.(kubetesting.DeleteAction).GetDeleteOptions().Preconditions
		return false, nil, nil // let the fake client's default reactor actually apply the delete
	})
	d := New(testDedupNamespace, "pod-x", clientset, 60*time.Second, noMaxHold)

	d.reapOnce(context.Background())

	assert.NotNil(t, gotPreconditions, "reaper's Delete must guard the List-then-Delete gap with Preconditions")
	assert.Equal(t, listed.UID, *gotPreconditions.UID)
	assert.Equal(t, listed.ResourceVersion, *gotPreconditions.ResourceVersion)
}

func TestReapOnceSkipsWhenConcurrentStealCausesConflict(t *testing.T) {
	old := metav1.NewTime(time.Now().Add(-time.Hour))
	holder := "pod-crashed"
	ttl := int32(60)
	staleLease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: "hpe-csi-dedup-stale", Namespace: testDedupNamespace, Labels: map[string]string{dedupLeaseLabelKey: dedupLeaseLabelValue}},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder, RenewTime: &metav1.MicroTime{Time: old.Time}, LeaseDurationSeconds: &ttl},
	}
	clientset := fake.NewSimpleClientset(staleLease)
	// Simulates another replica legitimately stealing this Lease (TryAcquire) in the gap between
	// the reaper's List and this Delete - the real server would reject the stale-precondition
	// Delete with Conflict instead of deleting the new holder's lock.
	clientset.PrependReactor("delete", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewConflict(coordinationv1.Resource("leases"), "hpe-csi-dedup-stale", assert.AnError)
	})
	d := New(testDedupNamespace, "pod-x", clientset, 60*time.Second, noMaxHold)

	assert.NotPanics(t, func() { d.reapOnce(context.Background()) })

	_, err := clientset.CoordinationV1().Leases(testDedupNamespace).Get(context.Background(), "hpe-csi-dedup-stale", metav1.GetOptions{})
	assert.NoError(t, err, "a Conflict from a concurrent steal must leave the new holder's lease intact")
}

func TestTryAcquireRenewsLeaseBeyondTTL(t *testing.T) {
	const ttl = 150 * time.Millisecond
	clientset := fake.NewSimpleClientset()
	holder := New(testDedupNamespace, "pod-a", clientset, ttl, noMaxHold)
	key := "CreateVolumeGroup:vg-1"

	assert.NoError(t, holder.TryAcquire(context.Background(), key))
	defer holder.stopRenewal(key)

	// Outlive the TTL several times over while the renewal goroutine (interval ttl/3) keeps
	// RenewTime fresh.
	time.Sleep(ttl * 4)

	name := leaseNameForKey(key)
	lease, err := clientset.CoordinationV1().Leases(testDedupNamespace).Get(context.Background(), name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, isLeaseStale(lease, ttl), "an actively-renewed lease must never look stale")

	// A second replica trying the same key must still see a genuine in-flight duplicate, not a
	// stale lease it can steal.
	other := New(testDedupNamespace, "pod-b", clientset, ttl, noMaxHold)
	err = other.TryAcquire(context.Background(), key)
	assert.ErrorIs(t, err, ErrDuplicateInFlight, "renewal must prevent a steal while the original holder is still alive")
}

func TestReleaseStopsRenewalGoroutine(t *testing.T) {
	const ttl = 100 * time.Millisecond
	clientset := fake.NewSimpleClientset()
	var updateCalls atomic.Int32
	clientset.PrependReactor("update", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		updateCalls.Add(1)
		return false, nil, nil // let the fake client's default reactor actually apply the update
	})
	d := New(testDedupNamespace, "pod-a", clientset, ttl, noMaxHold)
	key := "CreateVolumeGroup:vg-1"

	assert.NoError(t, d.TryAcquire(context.Background(), key))
	time.Sleep(ttl) // let at least one renewal tick happen
	assert.NoError(t, d.Release(context.Background(), key))

	countAfterRelease := updateCalls.Load()
	time.Sleep(ttl * 3) // several more renewal intervals' worth of time
	assert.Equal(t, countAfterRelease, updateCalls.Load(), "no further renewal Update calls must happen after Release")
}

func TestRenewOnceStopsWhenContextCancelled(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	d := New(testDedupNamespace, "pod-a", clientset, time.Minute, noMaxHold)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.False(t, d.renewOnce(ctx, "some-key", "some-lease-name"))
}

func TestRenewOnceStopsWhenLeaseNotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	d := New(testDedupNamespace, "pod-a", clientset, time.Minute, noMaxHold)

	assert.False(t, d.renewOnce(context.Background(), "some-key", "hpe-csi-dedup-missing"))
}

func TestRenewOnceStopsWhenNotOwner(t *testing.T) {
	name := leaseNameForKey("CreateVolumeGroup:vg-1")
	otherHolder := "pod-b"
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testDedupNamespace},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &otherHolder, AcquireTime: &metav1.MicroTime{Time: time.Now()}, RenewTime: &metav1.MicroTime{Time: time.Now()}},
	}
	clientset := fake.NewSimpleClientset(lease)
	clientset.PrependReactor("update", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		t.Fatal("renewOnce must not attempt to Update a lease it does not own")
		return false, nil, nil
	})
	d := New(testDedupNamespace, "pod-a", clientset, time.Minute, noMaxHold)

	assert.False(t, d.renewOnce(context.Background(), "CreateVolumeGroup:vg-1", name))
}

func TestRenewOnceStopsOnConflict(t *testing.T) {
	name := leaseNameForKey("CreateVolumeGroup:vg-1")
	holder := "pod-a"
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testDedupNamespace},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder, AcquireTime: &metav1.MicroTime{Time: time.Now()}, RenewTime: &metav1.MicroTime{Time: time.Now()}},
	}
	clientset := fake.NewSimpleClientset(lease)
	clientset.PrependReactor("update", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewConflict(coordinationv1.Resource("leases"), name, assert.AnError)
	})
	d := New(testDedupNamespace, holder, clientset, time.Minute, noMaxHold)

	assert.False(t, d.renewOnce(context.Background(), "CreateVolumeGroup:vg-1", name))
}

func TestRenewOnceRetriesOnTransientError(t *testing.T) {
	name := leaseNameForKey("CreateVolumeGroup:vg-1")
	holder := "pod-a"
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testDedupNamespace},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder, AcquireTime: &metav1.MicroTime{Time: time.Now()}, RenewTime: &metav1.MicroTime{Time: time.Now()}},
	}
	clientset := fake.NewSimpleClientset(lease)
	clientset.PrependReactor("update", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewServiceUnavailable("apiserver hiccup")
	})
	d := New(testDedupNamespace, holder, clientset, time.Minute, noMaxHold)

	assert.True(t, d.renewOnce(context.Background(), "CreateVolumeGroup:vg-1", name), "a transient error must be retried, not treated as lost ownership")
}

func TestRenewOnceStopsWhenMaxHoldDurationExceeded(t *testing.T) {
	name := leaseNameForKey("CreateVolumeGroup:vg-1")
	holder := "pod-a"
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testDedupNamespace},
		Spec: coordinationv1.LeaseSpec{
			HolderIdentity: &holder,
			AcquireTime:    &metav1.MicroTime{Time: time.Now().Add(-time.Hour)}, // held far longer than maxHoldDuration
			RenewTime:      &metav1.MicroTime{Time: time.Now()},                 // but still being actively renewed
		},
	}
	clientset := fake.NewSimpleClientset(lease)
	clientset.PrependReactor("update", "leases", func(action kubetesting.Action) (bool, runtime.Object, error) {
		t.Fatal("renewOnce must not renew past maxHoldDuration, even if still the owner")
		return false, nil, nil
	})
	d := New(testDedupNamespace, holder, clientset, time.Minute, 5*time.Minute)

	assert.False(t, d.renewOnce(context.Background(), "CreateVolumeGroup:vg-1", name),
		"a wedged handler that never calls Release must still let the lease go stale eventually")
}

func TestRenewOnceStopsWhenAcquireTimeMissing(t *testing.T) {
	name := leaseNameForKey("CreateVolumeGroup:vg-1")
	holder := "pod-a"
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testDedupNamespace},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder, RenewTime: &metav1.MicroTime{Time: time.Now()}}, // no AcquireTime
	}
	clientset := fake.NewSimpleClientset(lease)
	d := New(testDedupNamespace, holder, clientset, time.Minute, 5*time.Minute)

	assert.False(t, d.renewOnce(context.Background(), "CreateVolumeGroup:vg-1", name),
		"hold duration can't be verified without AcquireTime, so renewal must stop defensively")
}

func TestReleaseSkipsDeleteWhenNotOwner(t *testing.T) {
	key := "CreateVolumeGroup:vg-1"
	name := leaseNameForKey(key)
	otherHolder := "pod-b"
	lease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: testDedupNamespace},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &otherHolder, RenewTime: &metav1.MicroTime{Time: time.Now()}},
	}
	clientset := fake.NewSimpleClientset(lease)
	d := New(testDedupNamespace, "pod-a", clientset, time.Minute, noMaxHold)

	assert.NoError(t, d.Release(context.Background(), key), "Release must not error when another replica now legitimately owns the lease")

	_, err := clientset.CoordinationV1().Leases(testDedupNamespace).Get(context.Background(), name, metav1.GetOptions{})
	assert.NoError(t, err, "H2: Release must never delete a lease it does not own")
}

// deadlineRecordingClientset wraps a real clientset, recording whether each Lease API call it
// observes was given a bounded ctx (ctx.Deadline() set) - used to prove TryAcquire/Release/reapOnce
// never hand the caller's (possibly unbounded) context straight to the apiserver (M2).
type deadlineRecordingClientset struct {
	k8sclient.Interface
	mu        sync.Mutex
	deadlines []bool
}

func (c *deadlineRecordingClientset) record(ctx context.Context) {
	_, hasDeadline := ctx.Deadline()
	c.mu.Lock()
	c.deadlines = append(c.deadlines, hasDeadline)
	c.mu.Unlock()
}

func (c *deadlineRecordingClientset) CoordinationV1() coordinationv1client.CoordinationV1Interface {
	return &deadlineRecordingCoordinationV1{CoordinationV1Interface: c.Interface.CoordinationV1(), recorder: c}
}

type deadlineRecordingCoordinationV1 struct {
	coordinationv1client.CoordinationV1Interface
	recorder *deadlineRecordingClientset
}

func (c *deadlineRecordingCoordinationV1) Leases(namespace string) coordinationv1client.LeaseInterface {
	return &deadlineRecordingLeases{LeaseInterface: c.CoordinationV1Interface.Leases(namespace), recorder: c.recorder}
}

type deadlineRecordingLeases struct {
	coordinationv1client.LeaseInterface
	recorder *deadlineRecordingClientset
}

func (l *deadlineRecordingLeases) Create(ctx context.Context, lease *coordinationv1.Lease, opts metav1.CreateOptions) (*coordinationv1.Lease, error) {
	l.recorder.record(ctx)
	return l.LeaseInterface.Create(ctx, lease, opts)
}

func (l *deadlineRecordingLeases) Get(ctx context.Context, name string, opts metav1.GetOptions) (*coordinationv1.Lease, error) {
	l.recorder.record(ctx)
	return l.LeaseInterface.Get(ctx, name, opts)
}

func (l *deadlineRecordingLeases) Update(ctx context.Context, lease *coordinationv1.Lease, opts metav1.UpdateOptions) (*coordinationv1.Lease, error) {
	l.recorder.record(ctx)
	return l.LeaseInterface.Update(ctx, lease, opts)
}

func (l *deadlineRecordingLeases) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	l.recorder.record(ctx)
	return l.LeaseInterface.Delete(ctx, name, opts)
}

func (l *deadlineRecordingLeases) List(ctx context.Context, opts metav1.ListOptions) (*coordinationv1.LeaseList, error) {
	l.recorder.record(ctx)
	return l.LeaseInterface.List(ctx, opts)
}

func TestTryAcquireAndReleaseBoundCallsWithDeadline(t *testing.T) {
	recorder := &deadlineRecordingClientset{Interface: fake.NewSimpleClientset()}
	d := New(testDedupNamespace, "pod-a", recorder, time.Minute, noMaxHold)
	key := "CreateVolumeGroup:vg-1"

	assert.NoError(t, d.TryAcquire(context.Background(), key), "TryAcquire called with an unbounded context must still succeed")
	assert.NoError(t, d.Release(context.Background(), key), "Release called with an unbounded context must still succeed")

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	assert.NotEmpty(t, recorder.deadlines, "expected at least one Lease API call to be observed")
	for i, hasDeadline := range recorder.deadlines {
		assert.True(t, hasDeadline, "call #%d to the Lease API must be bounded by a deadline (M2), not rely on the caller's own context", i)
	}
}

func TestReapOnceBoundsCallsWithDeadline(t *testing.T) {
	old := metav1.NewTime(time.Now().Add(-time.Hour))
	holder := "pod-x"
	ttl := int32(60)
	staleLease := &coordinationv1.Lease{
		ObjectMeta: metav1.ObjectMeta{Name: "hpe-csi-dedup-stale", Namespace: testDedupNamespace, Labels: map[string]string{dedupLeaseLabelKey: dedupLeaseLabelValue}},
		Spec:       coordinationv1.LeaseSpec{HolderIdentity: &holder, RenewTime: &metav1.MicroTime{Time: old.Time}, LeaseDurationSeconds: &ttl},
	}
	recorder := &deadlineRecordingClientset{Interface: fake.NewSimpleClientset(staleLease)}
	d := New(testDedupNamespace, "pod-x", recorder, 60*time.Second, noMaxHold)

	d.reapOnce(context.Background())

	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	assert.NotEmpty(t, recorder.deadlines, "expected at least one Lease API call to be observed")
	for i, hasDeadline := range recorder.deadlines {
		assert.True(t, hasDeadline, "call #%d made by the reaper must be bounded by a deadline (M2), not rely on the caller's own context", i)
	}
}
