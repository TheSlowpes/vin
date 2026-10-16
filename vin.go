package vin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	MaxBatchSize   = 50
	DefaultTimeout = 30 * time.Second
)

// Client decodes VINs using NHTSA vPIC. Use NewClient to configure HTTP transport.
type Client struct {
	httpClient *http.Client
}

// NewClient uses a thirty-second HTTP timeout when httpClient is nil.
// A supplied client controls its own transport and timeout settings.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	return &Client{httpClient: httpClient}
}

var defaultClient = NewClient(nil)

// Decode decodes one VIN using the default HTTP timeout.
func Decode(vin string) (VIN, error) {
	return DecodeContext(context.Background(), vin)
}

// DecodeContext decodes one VIN and stops the request when ctx is cancelled.
func DecodeContext(ctx context.Context, vin string) (VIN, error) {
	return defaultClient.Decode(ctx, vin)
}

func (c *Client) Decode(ctx context.Context, vin string) (VIN, error) {
	params := url.Values{}
	params.Add("format", "json")
	requestURL := "https://vpic.nhtsa.dot.gov/api/vehicles/decodevinvalues/" + url.PathEscape(vin) + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return VIN{}, fmt.Errorf("create GET request: %w", err)
	}
	apiResponse, err := c.request(req)
	if err != nil {
		return VIN{}, err
	}

	if apiResponse.Count != 1 || len(apiResponse.Results) != 1 {
		return VIN{}, fmt.Errorf("unexpected number of results: count=%d, results=%d", apiResponse.Count, len(apiResponse.Results))
	}

	decoded := apiResponse.Results[0]
	return buildResponse(decoded)
}

func BatchDecode(vins []string) ([]VIN, error) {
	return BatchDecodeContext(context.Background(), vins)
}

// BatchDecodeContext decodes up to MaxBatchSize VINs using a cancellable request.
func BatchDecodeContext(ctx context.Context, vins []string) ([]VIN, error) {
	return defaultClient.BatchDecode(ctx, vins)
}

func (c *Client) BatchDecode(ctx context.Context, vins []string) ([]VIN, error) {
	if len(vins) == 0 {
		return []VIN{}, fmt.Errorf("no VINs provided for batch decoding")
	}
	if len(vins) > MaxBatchSize {
		return []VIN{}, fmt.Errorf("batch size exceeds maximum limit of %d", MaxBatchSize)
	}
	vinJoin := strings.Join(vins, ";")

	requestURL := "https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVINValuesBatch/"
	form := url.Values{}
	form.Add("format", "json")
	form.Add("data", vinJoin)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(form.Encode()))
	if err != nil {
		return []VIN{}, fmt.Errorf("create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	apiResponse, err := c.request(req)
	if err != nil {
		return []VIN{}, err
	}

	var vinsData []VIN
	for _, decoded := range apiResponse.Results {
		builtVIN, _ := buildResponse(decoded)
		vinsData = append(vinsData, builtVIN)
	}

	return vinsData, nil
}

func (c *Client) request(req *http.Request) (vinResponse, error) {
	client := c.httpClient
	if client == nil {
		client = defaultClient.httpClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return vinResponse{}, fmt.Errorf("execute %s request: %w", req.Method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return vinResponse{}, fmt.Errorf("vPIC request returned HTTP %d", resp.StatusCode)
	}
	var decoded vinResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return vinResponse{}, fmt.Errorf("decode response body: %w", err)
	}
	return decoded, nil
}

func (v VIN) String() string {
	return v.vin
}

type VINError struct {
	ErrorCode           string
	ErrorText           string
	AdditionalErrorText string
	SuggestedVIN        string
	PosssibleValues     string
}

func (e VINError) Error() string {
	return e.ErrorText
}

func buildResponse(decoded DecodedVIN) (VIN, error) {
	if decoded.ErrorCode != "0" {
		var err VINError
		err.ErrorCode = decoded.ErrorCode
		err.ErrorText = decoded.ErrorText
		err.AdditionalErrorText = decoded.AdditionalErrorText
		err.SuggestedVIN = decoded.SuggestedVIN
		err.PosssibleValues = decoded.PossibleValues
		return VIN{vin: decoded.VIN, Data: decoded, Error: err}, err
	}
	return VIN{vin: decoded.VIN, Data: decoded}, nil
}
