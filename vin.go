package vin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	MaxBatchSize = 50
)

func Decode(vin string) (VIN, error) {
	var apiResponse vinResponse

	client := &http.Client{}

	params := url.Values{}
	params.Add("format", "json")
	buitlURL := "https://vpic.nhtsa.dot.gov/api/vehicles/decodevinvalues/" + url.PathEscape(vin) + "?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, buitlURL, nil)
	if err != nil {
		return VIN{}, fmt.Errorf("create POST request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return VIN{}, fmt.Errorf("execute GET request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return VIN{}, fmt.Errorf("read response body: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		return VIN{}, fmt.Errorf("unmarshal response body: %w", err)
	}

	if apiResponse.Count != 1 {
		return VIN{}, fmt.Errorf("unexpected number of results: %d", apiResponse.Count)
	}

	decoded := apiResponse.Results[0]
	return buildResponse(decoded)
}

func BatchDecode(vins []string) ([]VIN, error) {
	if len(vins) == 0 {
		return []VIN{}, fmt.Errorf("no VINs provided for batch decoding")
	}
	if len(vins) > MaxBatchSize {
		return []VIN{}, fmt.Errorf("batch size exceeds maximum limit of %d", MaxBatchSize)
	}
	vinJoin := strings.Join(vins, ";")
	var apiResponse vinResponse

	client := &http.Client{}

	requestURL := "https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVINValuesBatch/"
	form := url.Values{}
	form.Add("format", "json")
	form.Add("data", vinJoin)

	req, err := http.NewRequest(http.MethodPost, requestURL, strings.NewReader(form.Encode()))
	if err != nil {
		return []VIN{}, fmt.Errorf("create POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return []VIN{}, fmt.Errorf("execute POST request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return []VIN{}, fmt.Errorf("read response body: %w", err)
	}

	if err := json.Unmarshal(bodyBytes, &apiResponse); err != nil {
		return []VIN{}, fmt.Errorf("unmarshal response body: %w", err)
	}

	var vinsData []VIN
	for _, decoded := range apiResponse.Results {
		builtVIN, _ := buildResponse(decoded)
		vinsData = append(vinsData, builtVIN)
	}

	return vinsData, nil
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
