package gridproxy

import (
	"fmt"
	"strconv"
	"strings"
)

func stringifyList(l []uint64) string {
	var ls []string
	for _, v := range l {
		ls = append(ls, strconv.FormatUint(v, 10))
	}
	return strings.Join(ls, ",")
}

func nodeParams(filter NodeFilter, limit Limit) string {

	var builder strings.Builder
	fmt.Fprintf(&builder, "?")

	if filter.Status != nil {
		fmt.Fprintf(&builder, "status=%s&", *filter.Status)
	}
	if filter.FreeMRU != nil && *filter.FreeMRU != 0 {
		fmt.Fprintf(&builder, "free_mru=%d&", *filter.FreeMRU)
	}
	if filter.FreeHRU != nil && *filter.FreeHRU != 0 {
		fmt.Fprintf(&builder, "free_hru=%d&", *filter.FreeHRU)
	}
	if filter.FreeSRU != nil && *filter.FreeSRU != 0 {
		fmt.Fprintf(&builder, "free_sru=%d&", *filter.FreeSRU)
	}
	if filter.Country != nil && *filter.Country != "" {
		fmt.Fprintf(&builder, "country=%s&", *filter.Country)
	}
	if filter.City != nil && *filter.City != "" {
		fmt.Fprintf(&builder, "city=%s&", *filter.City)
	}
	if filter.FarmName != nil && *filter.FarmName != "" {
		fmt.Fprintf(&builder, "farm_name=%s&", *filter.FarmName)
	}
	if filter.FarmIDs != nil && len(filter.FarmIDs) != 0 {
		fmt.Fprintf(&builder, "farm_ids=%s&", stringifyList(filter.FarmIDs))
	}
	if filter.FreeIPs != nil && *filter.FreeIPs != 0 {
		fmt.Fprintf(&builder, "free_ips=%d&", *filter.FreeIPs)
	}
	if filter.IPv4 != nil {
		fmt.Fprintf(&builder, "ipv4=%t&", *filter.IPv4)
	}
	if filter.IPv6 != nil {
		fmt.Fprintf(&builder, "ipv6=%t&", *filter.IPv6)
	}
	if filter.Domain != nil {
		fmt.Fprintf(&builder, "domain=%t&", *filter.Domain)
	}
	if filter.Rentable != nil {
		fmt.Fprintf(&builder, "rentable=%t&", *filter.Rentable)
	}
	if filter.RentedBy != nil {
		// passing 0 might be helpful to get available non-rented nodes
		fmt.Fprintf(&builder, "rented_by=%d&", *filter.RentedBy)
	}
	if filter.AvailableFor != nil {
		fmt.Fprintf(&builder, "available_for=%d&", *filter.AvailableFor)
	}
	if limit.Page != 0 {
		fmt.Fprintf(&builder, "page=%d&", limit.Page)
	}
	if limit.Size != 0 {
		fmt.Fprintf(&builder, "size=%d&", limit.Size)
	}
	res := builder.String()
	// pop the extra ? or &
	return res[:len(res)-1]
}

func farmParams(filter FarmFilter, limit Limit) string {

	var builder strings.Builder
	fmt.Fprintf(&builder, "?")

	if filter.FreeIPs != nil && *filter.FreeIPs != 0 {
		fmt.Fprintf(&builder, "free_ips=%d&", *filter.FreeIPs)
	}
	if filter.TotalIPs != nil && *filter.TotalIPs != 0 {
		fmt.Fprintf(&builder, "total_ips=%d&", *filter.TotalIPs)
	}
	if filter.StellarAddress != nil && *filter.StellarAddress != "" {
		fmt.Fprintf(&builder, "stellar_address=%s&", *filter.StellarAddress)
	}
	if filter.PricingPolicyID != nil {
		fmt.Fprintf(&builder, "pricing_policy_id=%d&", *filter.PricingPolicyID)
	}
	if filter.Version != nil {
		fmt.Fprintf(&builder, "version=%d&", *filter.Version)
	}
	if filter.FarmID != nil && *filter.FarmID != 0 {
		fmt.Fprintf(&builder, "farm_id=%d&", *filter.FarmID)
	}
	if filter.TwinID != nil && *filter.TwinID != 0 {
		fmt.Fprintf(&builder, "twin_id=%d&", *filter.TwinID)
	}
	if filter.Name != nil && *filter.Name != "" {
		fmt.Fprintf(&builder, "name=%s&", *filter.Name)
	}
	if filter.NameContains != nil && *filter.NameContains != "" {
		fmt.Fprintf(&builder, "name_contains=%s&", *filter.NameContains)
	}
	if filter.CertificationType != nil && *filter.CertificationType != "" {
		fmt.Fprintf(&builder, "certification_type=%s&", *filter.CertificationType)
	}
	if filter.Dedicated != nil {
		fmt.Fprintf(&builder, "dedicated=%t&", *filter.Dedicated)
	}
	if limit.Page != 0 {
		fmt.Fprintf(&builder, "page=%d&", limit.Page)
	}
	if limit.Size != 0 {
		fmt.Fprintf(&builder, "size=%d&", limit.Size)
	}

	res := builder.String()
	// pop the extra ? or &
	return res[:len(res)-1]
}
