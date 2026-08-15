package model

import "testing"

func TestIntegerEnergyConversions(t *testing.T) {
	energy, err := PowerToEnergyFloor(7_000, 15)
	if err != nil {
		t.Fatal(err)
	}
	if energy != 1_750 {
		t.Fatalf("energy=%d, want 1750", energy)
	}
	battery, err := ApplyChargeEfficiency(energy, 900_000)
	if err != nil {
		t.Fatal(err)
	}
	if battery != 1_575 {
		t.Fatalf("battery=%d, want 1575", battery)
	}
	grid, err := GridForChargeCeil(battery, 900_000)
	if err != nil {
		t.Fatal(err)
	}
	if grid != energy {
		t.Fatalf("round trip grid=%d, want %d", grid, energy)
	}
}

func TestMulDivRejectsOverflow(t *testing.T) {
	if _, err := MulDivFloor(1<<62, 8, 10); err == nil {
		t.Fatal("expected overflow error")
	}
}
