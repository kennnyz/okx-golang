package okx

import (
	"encoding/json"
	"testing"
)

func TestNumber(t *testing.T) {
	var v struct {
		A, B, C, D Number
	}
	if err := json.Unmarshal([]byte(`{"A":"0.1","B":12,"C":"","D":null}`), &v); err != nil {
		t.Fatal(err)
	}
	if v.A != "0.1" || v.B.Int64() != 12 || !v.C.IsZero() || v.D != "" {
		t.Fatalf("%+v", v)
	}
	if NumberFromFloat(0.1).String() != "0.1" || NumberFromFloat(65000).String() != "65000" {
		t.Fatal("NumberFromFloat")
	}
	b, _ := json.Marshal(struct{ P Number }{"1.50"})
	if string(b) != `{"P":"1.50"}` {
		t.Fatalf("marshal %s", b)
	}
}

func TestTime(t *testing.T) {
	var v struct{ A, B, C Time }
	if err := json.Unmarshal([]byte(`{"A":"1700000000123","B":1700000000123,"C":""}`), &v); err != nil {
		t.Fatal(err)
	}
	if v.A.UnixMilli() != 1700000000123 || !v.A.Equal(v.B.Time) || !v.C.IsZero() {
		t.Fatalf("%+v", v)
	}
	b, _ := json.Marshal(v)
	if string(b) != `{"A":"1700000000123","B":"1700000000123","C":""}` {
		t.Fatalf("marshal %s", b)
	}
}

func TestBoolAndInt(t *testing.T) {
	var v struct {
		A, B, C Bool
		D, E    Int
	}
	if err := json.Unmarshal([]byte(`{"A":"true","B":true,"C":"false","D":"7","E":8}`), &v); err != nil {
		t.Fatal(err)
	}
	if !v.A || !v.B || v.C || v.D != 7 || v.E != 8 {
		t.Fatalf("%+v", v)
	}
}

func TestBookLevelFormats(t *testing.T) {
	var levels []BookLevel
	if err := json.Unmarshal([]byte(`[["100.5","2","0","3"],["101","1","4"]]`), &levels); err != nil {
		t.Fatal(err)
	}
	if levels[0].Px != "100.5" || levels[0].Orders != 3 || levels[1].Orders != 4 {
		t.Fatalf("%+v", levels)
	}
}

func TestCandleFormats(t *testing.T) {
	var c []Candle
	in := `[["1700000000000","1","2","0.5","1.5","10","20","30","1"],["1700000000000","1","2","0.5","1.5","0"]]`
	if err := json.Unmarshal([]byte(in), &c); err != nil {
		t.Fatal(err)
	}
	if c[0].VolCcyQuote != "30" || !c[0].Confirmed || c[1].Vol != "" || c[1].Confirmed {
		t.Fatalf("%+v", c)
	}
}
