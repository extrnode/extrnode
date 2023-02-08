package solana

const MultipleValuesRequested = "multiple_values"

var FullMethodList = map[string]struct{}{
	"getAccountInfo":                    {},
	"getBalance":                        {},
	"getBlock":                          {},
	"getBlockHeight":                    {},
	"getBlockProduction":                {},
	"getBlockCommitment":                {},
	"getBlocks":                         {},
	"getBlocksWithLimit":                {},
	"getBlockTime":                      {},
	"getClusterNodes":                   {},
	"getEpochInfo":                      {},
	"getEpochSchedule":                  {},
	"getFeeForMessage":                  {},
	"getFirstAvailableBlock":            {},
	"getGenesisHash":                    {},
	"getHealth":                         {},
	"getHighestSnapshotSlot":            {},
	"getIdentity":                       {},
	"getInflationGovernor":              {},
	"getInflationRate":                  {},
	"getInflationReward":                {},
	"getLargestAccounts":                {},
	"getLatestBlockhash":                {},
	"getLeaderSchedule":                 {},
	"getMaxRetransmitSlot":              {},
	"getMaxShredInsertSlot":             {},
	"getMinimumBalanceForRentExemption": {},
	"getMultipleAccounts":               {},
	"getProgramAccounts":                {},
	"getRecentPerformanceSamples":       {},
	"getRecentPrioritizationFees":       {},
	"getSignaturesForAddress":           {},
	"getSignatureStatuses":              {},
	"getSlot":                           {},
	"getSlotLeader":                     {},
	"getSlotLeaders":                    {},
	"getStakeActivation":                {},
	"getStakeMinimumDelegation":         {},
	"getSupply":                         {},
	"getTokenAccountBalance":            {},
	"getTokenAccountsByDelegate":        {},
	"getTokenAccountsByOwner":           {},
	"getTokenLargestAccounts":           {},
	"getTokenSupply":                    {},
	"getTransaction":                    {},
	"getTransactionCount":               {},
	"getVersion":                        {},
	"getVoteAccounts":                   {},
	"isBlockhashValid":                  {},
	"minimumLedgerSlot":                 {},
	"requestAirdrop":                    {},
	"sendTransaction":                   {},
	"simulateTransaction":               {},

	// deprecated methods, but works now on solana mainnet
	"getConfirmedBlock":                 {},
	"getConfirmedBlocks":                {},
	"getConfirmedBlocksWithLimit":       {},
	"getConfirmedSignaturesForAddress2": {},
	"getConfirmedTransaction":           {},
	"getFeeCalculatorForBlockhash":      {},
	"getFeeRateGovernor":                {},
	"getFees":                           {},
	"getRecentBlockhash":                {},
	"getSnapshotSlot":                   {},
}

const (
	GetAccountInfo                    = "getAccountInfo"
	SendTransaction                   = "sendTransaction"
	GetSignaturesForAddress           = "getSignaturesForAddress"
	GetLatestBlockhash                = "getLatestBlockhash"
	GetSlot                           = "getSlot"
	GetTransaction                    = "getTransaction"
	GetInflationReward                = "getInflationReward"
	GetProgramAccounts                = "getProgramAccounts"
	GetSignatureStatuses              = "getSignatureStatuses"
	GetTokenAccountBalance            = "getTokenAccountBalance"
	GetMultipleAccounts               = "getMultipleAccounts"
	GetEpochInfo                      = "getEpochInfo"
	GetBalance                        = "getBalance"
	GetRecentPerformanceSamples       = "getRecentPerformanceSamples"
	GetVoteAccounts                   = "getVoteAccounts"
	GetInflationRate                  = "getInflationRate"
	GetSupply                         = "getSupply"
	GetBlockTime                      = "getBlockTime"
	GetBlockHeight                    = "getBlockHeight"
	GetMinimumBalanceForRentExemption = "getMinimumBalanceForRentExemption"
	IsBlockhashValid                  = "isBlockhashValid"
	GetTransactionCount               = "getTransactionCount"
	GetTokenAccountsByOwner           = "getTokenAccountsByOwner"
	GetBlock                          = "getBlock"
	GetVersion                        = "getVersion"
)
