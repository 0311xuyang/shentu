package v260_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	shentuapp "github.com/shentufoundation/shentu/v2/app"
	"github.com/shentufoundation/shentu/v2/common"
	v260 "github.com/shentufoundation/shentu/v2/x/gov/legacy/v260"
	"github.com/shentufoundation/shentu/v2/x/gov/types"
)

func Test_MigrateProposalStore(t *testing.T) {
	govKey := sdk.NewKVStoreKey(govtypes.StoreKey)
	ctx := testutil.DefaultContext(govKey, sdk.NewTransientStoreKey("transient_test"))
	cdc := shentuapp.MakeEncodingConfig().Marshaler
	store := ctx.KVStore(govKey)

	content := govtypes.NewTextProposal("title", "description")
	fmt.Println(content.ProposalRoute())
	msg, ok := content.(proto.Message)
	require.True(t, ok)
	contentAny, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)

	testsStatus := []struct {
		oldProposal v260.Proposal
	}{
		{
			v260.Proposal{ProposalId: 1, Status: 0, Content: contentAny},
		},
		{
			v260.Proposal{ProposalId: 2, Status: 1, Content: contentAny},
		},
		{
			v260.Proposal{ProposalId: 3, Status: 2, Content: contentAny},
		},
		{
			v260.Proposal{ProposalId: 4, Status: 3, Content: contentAny},
		},
		{
			v260.Proposal{ProposalId: 5, Status: 4, Content: contentAny},
		},
		{
			v260.Proposal{ProposalId: 6, Status: 5, Content: contentAny},
		},
		{
			v260.Proposal{ProposalId: 7, Status: 6, Content: contentAny},
		},
	}

	for _, test := range testsStatus {
		bz, err := cdc.Marshal(&test.oldProposal)
		require.NoError(t, err)
		store.Set(govtypes.ProposalKey(test.oldProposal.ProposalId), bz)
	}

	err = v260.MigrateProposalStore(ctx, govKey, cdc)
	require.NoError(t, err)
}

func Test_MigrateParams(t *testing.T) {
	var (
		depositParams govtypes.DepositParams
		tallyParams   govtypes.TallyParams
		customParams  types.CustomParams
	)

	app := shentuapp.Setup(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{Time: time.Now().UTC()})
	govSubspace := app.GetSubspace(govtypes.ModuleName)
	tableField := reflect.ValueOf(&govSubspace).Elem().FieldByName("table")
	tableFieldPtr := reflect.NewAt(tableField.Type(), unsafe.Pointer(tableField.UnsafeAddr()))
	tableFieldPtr.Elem().Set(reflect.ValueOf(v260.ParamKeyTable()))

	minInitialDepositTokens := sdk.TokensFromConsensusPower(1, sdk.DefaultPowerReduction)
	minDepositTokens := sdk.TokensFromConsensusPower(5, sdk.DefaultPowerReduction)
	defaultTally := govtypes.NewTallyParams(sdk.NewDecWithPrec(335, 3), sdk.NewDecWithPrec(6, 1), sdk.NewDecWithPrec(335, 3))
	certifierUpdateSecurityVoteTally := govtypes.NewTallyParams(sdk.NewDecWithPrec(335, 3), sdk.NewDecWithPrec(668, 3), sdk.NewDecWithPrec(335, 3))
	certifierUpdateStakeVoteTally := govtypes.NewTallyParams(sdk.NewDecWithPrec(335, 3), sdk.NewDecWithPrec(8, 1), sdk.NewDecWithPrec(335, 3))

	oldDepositParams := v260.DepositParams{
		MinInitialDeposit: sdk.Coins{sdk.NewCoin(common.MicroCTKDenom, minInitialDepositTokens)},
		MinDeposit:        sdk.Coins{sdk.NewCoin(common.MicroCTKDenom, minDepositTokens)},
		MaxDepositPeriod:  govtypes.DefaultPeriod,
	}
	oldTallyParams := v260.TallyParams{
		DefaultTally:                     &defaultTally,
		CertifierUpdateSecurityVoteTally: &certifierUpdateSecurityVoteTally,
		CertifierUpdateStakeVoteTally:    &certifierUpdateStakeVoteTally,
	}
	// set old data
	govSubspace.Set(ctx, govtypes.ParamStoreKeyDepositParams, &oldDepositParams)
	govSubspace.Set(ctx, govtypes.ParamStoreKeyTallyParams, &oldTallyParams)

	tableFieldPtr.Elem().Set(reflect.ValueOf(types.ParamKeyTable()))
	err := v260.MigrateParams(ctx, govSubspace)
	require.NoError(t, err)
	// get migrate params
	govSubspace.Get(ctx, govtypes.ParamStoreKeyDepositParams, &depositParams)
	govSubspace.Get(ctx, govtypes.ParamStoreKeyTallyParams, &tallyParams)
	govSubspace.Get(ctx, types.ParamStoreKeyCustomParams, &customParams)

	require.Equal(t, depositParams.MinDeposit, oldDepositParams.MinDeposit)
	require.Equal(t, depositParams.MaxDepositPeriod, oldDepositParams.MaxDepositPeriod)
	require.Equal(t, tallyParams, defaultTally)
	require.Equal(t, customParams.CertifierUpdateSecurityVoteTally, &certifierUpdateSecurityVoteTally)
	require.Equal(t, customParams.CertifierUpdateStakeVoteTally, &certifierUpdateStakeVoteTally)
}
