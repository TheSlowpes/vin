package vin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestClientPreservesVINCorrections(t *testing.T) {
	client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Query().Get("format") != "json" {
			t.Errorf("unexpected decode request: %s %s", req.Method, req.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"Count":1,"Results":[{"VIN":"1HGCM82633AO04352","ErrorCode":"5","ErrorText":"VIN has errors","SuggestedVIN":"1HGCM82633A004352","PossibleValues":"(12:0)","Make":"HONDA","Model":"Accord","ModelYear":"2003"}]}`))}, nil
	})})
	decoded, err := client.Decode(context.Background(), "1HGCM82633AO04352")
	var vinErr VINError
	if !errors.As(err, &vinErr) || vinErr.SuggestedVIN != testVIN || decoded.Data.Make != "HONDA" || decoded.Error == nil {
		t.Fatalf("VIN correction details lost: %#v, %v", decoded, err)
	}
}

func TestClientRequestsRespectCancellation(t *testing.T) {
	for _, batch := range []bool{false, true} {
		t.Run(map[bool]string{false: "single", true: "batch"}[batch], func(t *testing.T) {
			client := NewClient(&http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				<-req.Context().Done()
				return nil, req.Context().Err()
			})})
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var err error
			if batch {
				_, err = client.BatchDecode(ctx, []string{testVIN})
			} else {
				_, err = client.Decode(ctx, testVIN)
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("request did not preserve context cancellation: %v", err)
			}
		})
	}
}

func TestClientTimeoutAndResponseErrors(t *testing.T) {
	if got := NewClient(nil).httpClient.Timeout; got != 30*time.Second {
		t.Fatalf("default timeout = %s", got)
	}
	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{"HTTP failure", http.StatusServiceUnavailable, "upstream unavailable"},
		{"malformed JSON", http.StatusOK, "{"},
		{"missing result", http.StatusOK, `{"Count":1,"Results":[]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body))}, nil
			})})
			if _, err := client.Decode(context.Background(), testVIN); err == nil {
				t.Fatal("invalid upstream response should return an error")
			}
		})
	}
}
