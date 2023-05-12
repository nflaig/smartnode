package client

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	hexutil "github.com/rocket-pool/smartnode/shared/utils/hex"
)

// Request types
type VoluntaryExitMessage struct {
	Epoch          uinteger `json:"epoch"`
	ValidatorIndex string   `json:"validator_index"`
}
type VoluntaryExitRequest struct {
	Message   VoluntaryExitMessage `json:"message"`
	Signature byteArray            `json:"signature"`
}
type BLSToExecutionChangeMessage struct {
	ValidatorIndex     string    `json:"validator_index"`
	FromBLSPubkey      byteArray `json:"from_bls_pubkey"`
	ToExecutionAddress byteArray `json:"to_execution_address"`
}
type BLSToExecutionChangeRequest struct {
	Message   BLSToExecutionChangeMessage `json:"message"`
	Signature byteArray                   `json:"signature"`
}

// Response types
type SyncStatusResponse struct {
	Data struct {
		IsSyncing    bool     `json:"is_syncing"`
		HeadSlot     uinteger `json:"head_slot"`
		SyncDistance uinteger `json:"sync_distance"`
	} `json:"data"`
}
type Eth2ConfigResponse struct {
	Data struct {
		SecondsPerSlot               uinteger `json:"SECONDS_PER_SLOT"`
		SlotsPerEpoch                uinteger `json:"SLOTS_PER_EPOCH"`
		EpochsPerSyncCommitteePeriod uinteger `json:"EPOCHS_PER_SYNC_COMMITTEE_PERIOD"`
	} `json:"data"`
}
type Eth2DepositContractResponse struct {
	Data struct {
		ChainID uinteger       `json:"chain_id"`
		Address common.Address `json:"address"`
	} `json:"data"`
}
type GenesisResponse struct {
	Data struct {
		GenesisTime           uinteger  `json:"genesis_time"`
		GenesisForkVersion    byteArray `json:"genesis_fork_version"`
		GenesisValidatorsRoot byteArray `json:"genesis_validators_root"`
	} `json:"data"`
}
type FinalityCheckpointsResponse struct {
	Data struct {
		PreviousJustified struct {
			Epoch uinteger `json:"epoch"`
		} `json:"previous_justified"`
		CurrentJustified struct {
			Epoch uinteger `json:"epoch"`
		} `json:"current_justified"`
		Finalized struct {
			Epoch uinteger `json:"epoch"`
		} `json:"finalized"`
	} `json:"data"`
}
type ForkResponse struct {
	Data struct {
		PreviousVersion byteArray `json:"previous_version"`
		CurrentVersion  byteArray `json:"current_version"`
		Epoch           uinteger  `json:"epoch"`
	} `json:"data"`
}
type AttestationsResponse struct {
	Data []Attestation `json:"data"`
}
type BeaconBlockResponse struct {
	Data struct {
		Message struct {
			Slot          uinteger `json:"slot"`
			ProposerIndex string   `json:"proposer_index"`
			Body          struct {
				Eth1Data struct {
					DepositRoot  byteArray `json:"deposit_root"`
					DepositCount uinteger  `json:"deposit_count"`
					BlockHash    byteArray `json:"block_hash"`
				} `json:"eth1_data"`
				Attestations     []Attestation `json:"attestations"`
				ExecutionPayload *struct {
					FeeRecipient byteArray `json:"fee_recipient"`
					BlockNumber  uinteger  `json:"block_number"`
				} `json:"execution_payload"`
			} `json:"body"`
		} `json:"message"`
	} `json:"data"`
}
type ValidatorsResponse struct {
	Data []Validator `json:"data"`
}
type Validator struct {
	Index     string   `json:"index"`
	Balance   uinteger `json:"balance"`
	Status    string   `json:"status"`
	Validator struct {
		Pubkey                     byteArray `json:"pubkey"`
		WithdrawalCredentials      byteArray `json:"withdrawal_credentials"`
		EffectiveBalance           uinteger  `json:"effective_balance"`
		Slashed                    bool      `json:"slashed"`
		ActivationEligibilityEpoch uinteger  `json:"activation_eligibility_epoch"`
		ActivationEpoch            uinteger  `json:"activation_epoch"`
		ExitEpoch                  uinteger  `json:"exit_epoch"`
		WithdrawableEpoch          uinteger  `json:"withdrawable_epoch"`
	} `json:"validator"`
}
type SyncDutiesResponse struct {
	Data []SyncDuty `json:"data"`
}
type SyncDuty struct {
	Pubkey               byteArray  `json:"pubkey"`
	ValidatorIndex       string     `json:"validator_index"`
	SyncCommitteeIndices []uinteger `json:"validator_sync_committee_indices"`
}
type ProposerDutiesResponse struct {
	Data []ProposerDuty `json:"data"`
}
type ProposerDuty struct {
	ValidatorIndex string `json:"validator_index"`
}

type CommitteesResponse struct {
	Data []Committee `json:"data"`
}

type Committee struct {
	Index      uinteger `json:"index"`
	Slot       uinteger `json:"slot"`
	Validators []string `json:"validators"`
}

// Custom deserialization logic for Committee allows us to pool the validator
// slices for reuse. They're quite large, so this cuts down on allocations
// substantially.
var validatorSlicePool sync.Pool = sync.Pool{
	New: func() any {
		return make([]string, 0, 1024)
	},
}

func (c *Committee) UnmarshalJSON(body []byte) error {
	var committee map[string]*json.RawMessage

	pooledSlice := validatorSlicePool.Get().([]string)

	c.Validators = pooledSlice

	// Partially parse the json
	if err := json.Unmarshal(body, &committee); err != nil {
		return fmt.Errorf("error unmarshalling committee json: %w\n", err)
	}

	// Parse each field
	if err := json.Unmarshal(*committee["index"], &c.Index); err != nil {
		return err
	}
	if err := json.Unmarshal(*committee["slot"], &c.Slot); err != nil {
		return err
	}
	// Since c.Validators was preallocated, this will re-use a buffer if one was available.
	if err := json.Unmarshal(*committee["validators"], &c.Validators); err != nil {
		return err
	}

	return nil
}

func (c *CommitteesResponse) Count() int {
	return len(c.Data)
}

func (c *CommitteesResponse) Index(idx int) uint64 {
	return uint64(c.Data[idx].Index)
}

func (c *CommitteesResponse) Slot(idx int) uint64 {
	return uint64(c.Data[idx].Slot)
}

func (c *CommitteesResponse) Validators(idx int) []string {
	return c.Data[idx].Validators
}

func (c *CommitteesResponse) Release() {
	for _, committee := range c.Data {
		// Reset the slice length to 0 (capacity stays the same)
		committee.Validators = committee.Validators[:0]
		// Return the slice for reuse
		validatorSlicePool.Put(committee.Validators)
	}
}

type Attestation struct {
	AggregationBits string `json:"aggregation_bits"`
	Data            struct {
		Slot  uinteger `json:"slot"`
		Index uinteger `json:"index"`
	} `json:"data"`
}

// Unsigned integer type
type uinteger uint64

func (i uinteger) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.Itoa(int(i)))
}
func (i *uinteger) UnmarshalJSON(data []byte) error {

	// Unmarshal string
	var dataStr string
	if err := json.Unmarshal(data, &dataStr); err != nil {
		return err
	}

	// Parse integer value
	value, err := strconv.ParseUint(dataStr, 10, 64)
	if err != nil {
		return err
	}

	// Set value and return
	*i = uinteger(value)
	return nil

}

// Byte array type
type byteArray []byte

func (b byteArray) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.AddPrefix(hex.EncodeToString(b)))
}
func (b *byteArray) UnmarshalJSON(data []byte) error {

	// Unmarshal string
	var dataStr string
	if err := json.Unmarshal(data, &dataStr); err != nil {
		return err
	}

	// Decode hex
	value, err := hex.DecodeString(hexutil.RemovePrefix(dataStr))
	if err != nil {
		return err
	}

	// Set value and return
	*b = value
	return nil

}
