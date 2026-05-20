package vin

import (
	"errors"
	"strings"
	"testing"
)

const testVIN = "1HGCM82633A004352"

func TestDecodeValidVIN(t *testing.T) {
	got, err := Decode(testVIN)
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if got.String() != testVIN {
		t.Fatalf("Decode().String() = %q, want %q", got.String(), testVIN)
	}
	if got.Data.VIN != testVIN {
		t.Fatalf("Decode().Data.VIN = %q, want %q", got.Data.VIN, testVIN)
	}
	if got.Data.Make == "" {
		t.Fatal("Decode().Data.Make is empty")
	}
	if got.Data.ModelYear == "" {
		t.Fatal("Decode().Data.ModelYear is empty")
	}
	if got.Error != nil {
		t.Fatalf("Decode().Error = %v, want nil", got.Error)
	}
}

func TestDecodeInvalidVINReturnsVINError(t *testing.T) {
	got, err := Decode("not-a-vin")
	if err == nil {
		t.Fatal("Decode() error = nil, want VINError")
	}

	var vinErr VINError
	if !errors.As(err, &vinErr) {
		t.Fatalf("Decode() error type = %T, want VINError", err)
	}
	if vinErr.ErrorCode == "" || vinErr.ErrorCode == "0" {
		t.Fatalf("VINError.ErrorCode = %q, want non-zero code", vinErr.ErrorCode)
	}
	if got.Error == nil {
		t.Fatal("Decode().Error = nil, want VINError")
	}
	if got.String() != "not-a-vin" {
		t.Fatalf("Decode().String() = %q, want input VIN", got.String())
	}
}

func TestBatchDecodeValidation(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got, err := BatchDecode(nil)
		if err == nil {
			t.Fatal("BatchDecode() error = nil, want error")
		}
		if got == nil {
			t.Fatal("BatchDecode() result = nil, want empty slice")
		}
		if !strings.Contains(err.Error(), "no VINs") {
			t.Fatalf("BatchDecode() error = %q, want no VINs message", err.Error())
		}
	})

	t.Run("too many", func(t *testing.T) {
		vins := make([]string, MaxBatchSize+1)
		got, err := BatchDecode(vins)
		if err == nil {
			t.Fatal("BatchDecode() error = nil, want error")
		}
		if got == nil {
			t.Fatal("BatchDecode() result = nil, want empty slice")
		}
		if !strings.Contains(err.Error(), "batch size exceeds") {
			t.Fatalf("BatchDecode() error = %q, want batch size message", err.Error())
		}
	})
}

func TestBatchDecodeSingleVIN(t *testing.T) {
	got, err := BatchDecode([]string{testVIN})
	if err != nil {
		t.Fatalf("BatchDecode() error = %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("len(BatchDecode()) = %d, want 1", len(got))
	}
	if got[0].String() != testVIN {
		t.Fatalf("BatchDecode()[0].String() = %q, want %q", got[0].String(), testVIN)
	}
	if got[0].Data.Make == "" {
		t.Fatal("BatchDecode()[0].Data.Make is empty")
	}
}

func TestBatchDecodeIncludesErrorsPerVIN(t *testing.T) {
	got, err := BatchDecode([]string{testVIN, "not-a-vin"})
	if err != nil {
		t.Fatalf("BatchDecode() error = %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(BatchDecode()) = %d, want 2", len(got))
	}
	if got[0].Error != nil {
		t.Fatalf("BatchDecode()[0].Error = %v, want nil", got[0].Error)
	}
	if got[1].Error == nil {
		t.Fatal("BatchDecode()[1].Error = nil, want VINError")
	}

	var vinErr VINError
	if !errors.As(got[1].Error, &vinErr) {
		t.Fatalf("BatchDecode()[1].Error type = %T, want VINError", got[1].Error)
	}
	if vinErr.ErrorCode == "" || vinErr.ErrorCode == "0" {
		t.Fatalf("VINError.ErrorCode = %q, want non-zero code", vinErr.ErrorCode)
	}
}
