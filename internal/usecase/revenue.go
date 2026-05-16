package usecase

// RevenueSplit holds the result of a revenue split calculation
type RevenueSplit struct {
	HostEarning      int64
	AgencyCommission int64
	PlatformFee      int64
}

// DefaultPlatformFeePercent is the platform's share
const DefaultPlatformFeePercent = 20

// CalculateRevenueSplit computes the revenue split for a gift transaction.
//   - totalPrice: the total gift value in IDR
//   - hasAgency: whether the host is under an agency
//   - agencyCommissionRate: the agency's commission % (e.g. 20 means 20%)
//
// Rules:
//
//	With Agency:  Host 60%, Agency 20%, Platform 20%
//	Without Agency: Host 80%, Platform 20%
//
// The agencyCommissionRate is used to determine agency share from the "agency+host" pool (80%).
// Platform always gets 20%.
func CalculateRevenueSplit(totalPrice int64, hasAgency bool, agencyCommissionRate int) RevenueSplit {
	platformFee := totalPrice * int64(DefaultPlatformFeePercent) / 100
	remaining := totalPrice - platformFee // 80% of total

	if !hasAgency {
		return RevenueSplit{
			HostEarning:      remaining,
			AgencyCommission: 0,
			PlatformFee:      platformFee,
		}
	}

	// With agency: split the remaining 80% between host and agency
	agencyCommission := remaining * int64(agencyCommissionRate) / 100
	hostEarning := remaining - agencyCommission

	return RevenueSplit{
		HostEarning:      hostEarning,
		AgencyCommission: agencyCommission,
		PlatformFee:      platformFee,
	}
}
