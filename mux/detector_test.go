package mux

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	krakend "github.com/krakendio/krakend-botdetector/v2/krakend"
	"github.com/luraproject/lura/v2/config"
	"github.com/luraproject/lura/v2/logging"
	"github.com/luraproject/lura/v2/proxy"
	luramux "github.com/luraproject/lura/v2/router/mux"
)

func TestRegister(t *testing.T) {
	cfg := config.ServiceConfig{
		ExtraConfig: config.ExtraConfig{
			krakend.Namespace: map[string]interface{}{
				"deny":  []interface{}{"a", "b"},
				"allow": []interface{}{"c", "Pingdom.com_bot_version_1.1"},
				"patterns": []interface{}{
					`(Pingdom.com_bot_version_)(\d+)\.(\d+)`,
					`(facebookexternalhit)/(\d+)\.(\d+)`,
				},
			},
		},
	}

	middleware := NewMiddleware(cfg.ExtraConfig, logging.NoOp)

	mux := http.NewServeMux()
	mux.Handle("/", middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "")
	})))

	if err := testDetection(mux); err != nil {
		t.Error(err)
	}
}

func TestNew(t *testing.T) {
	cfg := &config.EndpointConfig{
		Method: "GET",
		ExtraConfig: config.ExtraConfig{
			krakend.Namespace: map[string]interface{}{
				"deny":  []interface{}{"a", "b"},
				"allow": []interface{}{"c", "Pingdom.com_bot_version_1.1"},
				"patterns": []interface{}{
					`(Pingdom.com_bot_version_)(\d+)\.(\d+)`,
					`(facebookexternalhit)/(\d+)\.(\d+)`,
				},
			},
		},
	}

	proxyfunc := func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		return &proxy.Response{
			IsComplete: true,
			Data:       map[string]interface{}{"": ""},
		}, nil
	}

	mux := http.NewServeMux()
	mux.Handle("/", New(luramux.EndpointHandler, logging.NoOp)(cfg, proxyfunc))

	if err := testDetection(mux); err != nil {
		t.Error(err)
	}
}

func testDetection(muxServer http.Handler) error {
	ts := httptest.NewServer(muxServer)
	defer ts.Close()

	for i, ua := range []string{
		"abcd",
		"",
		"c",
		"Pingdom.com_bot_version_1.1",
	} {

		url := ts.URL + "/"
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Add("User-Agent", ua)

		w := httptest.NewRecorder()
		muxServer.ServeHTTP(w, req)

		if w.Result().StatusCode != 200 {
			return fmt.Errorf("the req #%d has been detected as a bot: %s", i, ua)
		}
	}

	for i, ua := range []string{
		"a",
		"b",
		"facebookexternalhit/1.1",
		"Pingdom.com_bot_version_1.2",
	} {
		req, _ := http.NewRequest("GET", ts.URL+"/", nil)
		req.Header.Add("User-Agent", ua)

		w := httptest.NewRecorder()
		muxServer.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusForbidden {
			return fmt.Errorf("the req #%d has not been detected as a bot: %s", i, ua)
		}
	}
	return nil
}
