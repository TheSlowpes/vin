# vin

Go client for decoding Vehicle Identification Numbers using the NHTSA VPIC API.

This package currently focuses on VIN decoding. Support for other vPIC endpoints is planned for future releases.

## Install

```sh
go get github.com/TheSlowpes/vin
```

## Usage

Decode one VIN:

```go
package main

import (
	"fmt"
	"log"

	"github.com/TheSlowpes/vin"
)

func main() {
	decoded, err := vin.Decode("1HGCM82633A004352")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(decoded.String())
	fmt.Println(decoded.Data.Make)
	fmt.Println(decoded.Data.Model)
	fmt.Println(decoded.Data.ModelYear)
}
```

Decode multiple VINs:

```go
decoded, err := vin.BatchDecode([]string{
	"1HGCM82633A004352",
	"1FAFP404X1F123456",
})
if err != nil {
	log.Fatal(err)
}

for _, vehicle := range decoded {
	if vehicle.Error != nil {
		fmt.Printf("%s: %v\n", vehicle.String(), vehicle.Error)
		continue
	}

	fmt.Printf("%s: %s %s %s\n", vehicle.String(), vehicle.Data.ModelYear, vehicle.Data.Make, vehicle.Data.Model)
}
```

## API

### `Decode(vin string) (VIN, error)`

Decodes one VIN. If NHTSA reports a VIN-specific decode error, the returned error can be inspected as `VINError`.

### `BatchDecode(vins []string) ([]VIN, error)`

Decodes up to `MaxBatchSize` VINs in one request. Transport or request-level errors are returned as the function error. VIN-specific decode errors are stored on each returned `VIN` in the `Error` field.

### `VIN`

`VIN` contains the decoded response data from NHTSA:

```go
type VIN struct {
	Data  DecodedVIN
	Error error
}
```

Use `String()` to get the original VIN value.

## Notes

This package calls the public NHTSA VPIC API at `https://vpic.nhtsa.dot.gov/api/`. Tests also call the live API, so they require network access and can fail if the upstream service is unavailable.
