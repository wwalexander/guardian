package guardian

import (
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestRouteLimitProvider(t *testing.T) {
	fooBarRouteLimit := Limit{Count: 2, Duration: time.Minute, Enabled: true}
	route := url.URL{Path: "/foo/bar"}
	routeLimits := map[url.URL]Limit{route: fooBarRouteLimit}
	globalLimit := Limit{Count: 2, Duration: time.Minute, Enabled: true}
	cs, s := newTestConfStoreWithDefaults(t, nil, nil, globalLimit, false)
	defer s.Close()

	cs.SetRouteRateLimits(routeLimits)
	cs.UpdateCachedConf()

	tests := []struct {
		name      string
		req       Request
		wantLimit Limit
	}{
		{
			name:      "route with limit",
			req:       Request{Path: "/foo/bar"},
			wantLimit: fooBarRouteLimit,
		},
		{
			name:      "sub route without limit",
			req:       Request{Path: "/foo/bar/baz"},
			wantLimit: Limit{Enabled: false},
		},
		{
			name:      "route without limit",
			req:       Request{Path: "/baz"},
			wantLimit: Limit{Enabled: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rlp := NewRouteRateLimitProvider(cs, TestingLogger)
			if gotLimit := rlp.GetLimit(tt.req); !reflect.DeepEqual(gotLimit, tt.wantLimit) {
				t.Errorf("GetLimit() = %v, want %v", gotLimit, tt.wantLimit)
			}
		})
	}
}

func TestRouteLimitProviderUpdates(t *testing.T) {
	fooBarRouteLimit := Limit{Count: 2, Duration: time.Minute, Enabled: true}
	route := url.URL{Path: "/foo/bar"}
	routeLimits := map[url.URL]Limit{route: fooBarRouteLimit}
	globalLimit := Limit{Count: 2, Duration: time.Minute, Enabled: true}
	cs, s := newTestConfStoreWithDefaults(t, nil, nil, globalLimit, false)
	defer s.Close()

	cs.SetRouteRateLimits(routeLimits)
	cs.UpdateCachedConf()

	rlp := NewRouteRateLimitProvider(cs, TestingLogger)
	gotLimit := rlp.GetLimit(Request{Path: "/foo/bar"})
	if !reflect.DeepEqual(gotLimit, fooBarRouteLimit) {
		t.Errorf("GetLimit() = %v, want %v", gotLimit, fooBarRouteLimit)
	}

	fooBarRouteLimit = Limit{Count: 43, Duration: time.Minute, Enabled: true}

	newRouteLimits := map[url.URL]Limit{route: fooBarRouteLimit}
	cs.SetRouteRateLimits(newRouteLimits)
	cs.UpdateCachedConf()

	gotLimit = rlp.GetLimit(Request{Path: "/foo/bar"})
	if !reflect.DeepEqual(gotLimit, fooBarRouteLimit) {
		t.Errorf("GetLimit() = %v, want %v", gotLimit, fooBarRouteLimit)
	}
}
