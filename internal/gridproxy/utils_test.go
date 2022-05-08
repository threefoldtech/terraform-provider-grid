package gridproxy

import "testing"

func nodesFilterValues() (NodeFilter, Limit, string) {
	Up := "up"
	Egypt := "Egypt"
	Mansoura := "Mansoura"
	Freefarm := "Freefarm"
	trueVal := true
	falseVal := false
	ints := []uint64{0, 1, 2, 3, 4, 5, 6}
	f := NodeFilter{
		Status:       &Up,
		FreeMRU:      &ints[1],
		FreeHRU:      &ints[2],
		FreeSRU:      &ints[3],
		Country:      &Egypt,
		City:         &Mansoura,
		FarmName:     &Freefarm,
		FarmIDs:      []uint64{1, 2},
		FreeIPs:      &ints[4],
		IPv4:         &trueVal,
		IPv6:         &falseVal,
		Domain:       &trueVal,
		Rentable:     &falseVal,
		RentedBy:     &ints[5],
		AvailableFor: &ints[6],
	}
	l := Limit{
		Page: 12,
		Size: 13,
	}
	return f, l, "?status=up&free_mru=1&free_hru=2&free_sru=3&country=Egypt&city=Mansoura&farm_name=Freefarm&farm_ids=1,2&free_ips=4&ipv4=true&ipv6=false&domain=true&rentable=false&rented_by=5&available_for=6&page=12&size=13"
}

func farmsFilterValues() (FarmFilter, Limit, string) {
	StellarAddress := "StellarAddress"
	FreeFarm := "freefarm"
	FreeFar := "freefar"
	DYI := "DYI"
	Dedicated := false
	ints := []uint64{0, 1, 2, 3, 4, 5, 6}
	f := FarmFilter{
		FreeIPs:           &ints[1],
		TotalIPs:          &ints[2],
		StellarAddress:    &StellarAddress,
		PricingPolicyID:   &ints[3],
		Version:           &ints[4],
		FarmID:            &ints[5],
		TwinID:            &ints[6],
		Name:              &FreeFarm,
		NameContains:      &FreeFar,
		CertificationType: &DYI,
		Dedicated:         &Dedicated,
	}
	l := Limit{
		Page: 12,
		Size: 13,
	}

	return f, l, "?free_ips=1&total_ips=2&stellar_address=StellarAddress&pricing_policy_id=3&version=4&farm_id=5&twin_id=6&name=freefarm&name_contains=freefar&certification_type=DYI&dedicated=false&page=12&size=13"
}

func TestNodeFilter(t *testing.T) {
	f, l, expected := nodesFilterValues()
	found := nodeParams(f, l)
	if found != expected {
		t.Fatalf("found: %s, expected: %s", found, expected)
	}
}

func TestEmptyNodeFilter(t *testing.T) {
	found := nodeParams(NodeFilter{}, Limit{})
	expected := ""
	if found != expected {
		t.Fatalf("found: %s, expected: %s", found, expected)
	}
}

func TestFarmFilter(t *testing.T) {
	f, l, expected := farmsFilterValues()
	found := farmParams(f, l)
	if found != expected {
		t.Fatalf("found: %s, expected: %s", found, expected)
	}
}

func TestEmptyFarmFilter(t *testing.T) {
	found := nodeParams(NodeFilter{}, Limit{})
	expected := ""
	if found != expected {
		t.Fatalf("found: %s, expected: %s", found, expected)
	}
}
