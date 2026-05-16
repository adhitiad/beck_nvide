package usecase

import "testing"

func TestCalculateRevenueSplit_WithAgency(t *testing.T) {
	// Total 100,000 IDR gift
	// Platform: 20% = 20,000
	// Remaining: 80,000
	// Agency commission rate: 25% of remaining = 20,000
	// Host: 80,000 - 20,000 = 60,000
	split := CalculateRevenueSplit(100000, true, 25)

	if split.PlatformFee != 20000 {
		t.Errorf("Expected platform fee 20000, got %d", split.PlatformFee)
	}
	if split.AgencyCommission != 20000 {
		t.Errorf("Expected agency commission 20000, got %d", split.AgencyCommission)
	}
	if split.HostEarning != 60000 {
		t.Errorf("Expected host earning 60000, got %d", split.HostEarning)
	}

	// Verify total adds up
	total := split.PlatformFee + split.AgencyCommission + split.HostEarning
	if total != 100000 {
		t.Errorf("Expected total 100000, got %d", total)
	}
}

func TestCalculateRevenueSplit_WithoutAgency(t *testing.T) {
	// Total 100,000 IDR gift
	// Platform: 20% = 20,000
	// Host: 80% = 80,000
	// Agency: 0
	split := CalculateRevenueSplit(100000, false, 0)

	if split.PlatformFee != 20000 {
		t.Errorf("Expected platform fee 20000, got %d", split.PlatformFee)
	}
	if split.AgencyCommission != 0 {
		t.Errorf("Expected agency commission 0, got %d", split.AgencyCommission)
	}
	if split.HostEarning != 80000 {
		t.Errorf("Expected host earning 80000, got %d", split.HostEarning)
	}

	total := split.PlatformFee + split.HostEarning
	if total != 100000 {
		t.Errorf("Expected total 100000, got %d", total)
	}
}

func TestCalculateRevenueSplit_SmallAmount(t *testing.T) {
	// 100 IDR (minimum gift)
	split := CalculateRevenueSplit(100, true, 25)

	if split.PlatformFee != 20 {
		t.Errorf("Expected platform fee 20, got %d", split.PlatformFee)
	}
	// Remaining: 80. Agency 25% of 80 = 20. Host = 60.
	if split.AgencyCommission != 20 {
		t.Errorf("Expected agency commission 20, got %d", split.AgencyCommission)
	}
	if split.HostEarning != 60 {
		t.Errorf("Expected host earning 60, got %d", split.HostEarning)
	}
}

func TestCalculateRevenueSplit_HighCommission(t *testing.T) {
	// Agency commission 30% (max)
	split := CalculateRevenueSplit(1000000, true, 30)

	if split.PlatformFee != 200000 {
		t.Errorf("Expected platform fee 200000, got %d", split.PlatformFee)
	}
	// Remaining: 800,000. Agency 30% of 800,000 = 240,000. Host = 560,000.
	if split.AgencyCommission != 240000 {
		t.Errorf("Expected agency commission 240000, got %d", split.AgencyCommission)
	}
	if split.HostEarning != 560000 {
		t.Errorf("Expected host earning 560000, got %d", split.HostEarning)
	}

	total := split.PlatformFee + split.AgencyCommission + split.HostEarning
	if total != 1000000 {
		t.Errorf("Expected total 1000000, got %d", total)
	}
}
