package reqcontext_test

import (
	"context"
	"testing"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/reqcontext"
	"github.com/stretchr/testify/require"
)

func TestRequestIDRoundTrip(t *testing.T) {
	ctx := reqcontext.WithRequestID(context.Background(), "abc-123")
	require.Equal(t, "abc-123", reqcontext.RequestIDFromContext(ctx))
}

func TestRequestIDMissing(t *testing.T) {
	require.Equal(t, "", reqcontext.RequestIDFromContext(context.Background()))
}

func TestTimingAccumulates(t *testing.T) {
	ctx := reqcontext.WithTiming(context.Background())

	reqcontext.AddServiceTime(ctx, 5*time.Millisecond)
	reqcontext.AddServiceTime(ctx, 3*time.Millisecond)
	reqcontext.AddRepoTime(ctx, 2*time.Millisecond)

	serviceMS, repoMS := reqcontext.TimingFromContext(ctx)
	require.Equal(t, int64(8), serviceMS)
	require.Equal(t, int64(2), repoMS)
}

func TestTrackHelpersRecordElapsed(t *testing.T) {
	ctx := reqcontext.WithTiming(context.Background())

	func() {
		defer reqcontext.TrackService(ctx)()
		time.Sleep(2 * time.Millisecond)
	}()

	stopRepo := reqcontext.TrackRepo(ctx)
	time.Sleep(2 * time.Millisecond)
	stopRepo()

	serviceMS, repoMS := reqcontext.TimingFromContext(ctx)
	require.GreaterOrEqual(t, serviceMS, int64(1))
	require.GreaterOrEqual(t, repoMS, int64(1))
}

func TestTimingNoAccumulatorIsNoop(t *testing.T) {
	ctx := context.Background()
	require.NotPanics(t, func() {
		reqcontext.AddServiceTime(ctx, time.Second)
		reqcontext.AddRepoTime(ctx, time.Second)
	})

	serviceMS, repoMS := reqcontext.TimingFromContext(ctx)
	require.Zero(t, serviceMS)
	require.Zero(t, repoMS)
}
