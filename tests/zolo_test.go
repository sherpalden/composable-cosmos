package tests

import (
	"github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"gotest.tools/v3/assert"
	"testing"
)

func TestGovModuleAddr(t *testing.T) {
	govAddr := types.MustBech32ifyAddressBytes("centauri", authtypes.NewModuleAddress(govtypes.ModuleName))
	assert.Equal(t, govAddr, "centauri10d07y265gmmuvt4z0w9aw880jnsr700j7g7ejq", "they should be equal")
}
