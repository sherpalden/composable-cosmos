package tests

import (
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"gotest.tools/v3/assert"
	"testing"
)

func TestGovModuleAddr(t *testing.T) {
	govAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	//govAddr = cosmos10d07y265gmmuvt4z0w9aw880jnsr700j6zn9kn
	assert.Equal(t, govAddr, "centauri10556m38z4x6pqalr9rl5ytf3cff8q46nk85k9m", "they should be equal")
}
