package cli

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/notional-labs/centauri/v3/x/transfermiddleware/types"
	"github.com/spf13/cobra"
)

// GetTxCmd returns the tx commands for router
func GetTxCmd() *cobra.Command {
	txCmd := &cobra.Command{
		Use:                        "transfermiddleware",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		Short:                      "Registry and remove IBC dotsama chain infomation",
		Long:                       "Registry and remove IBC dotsama chain infomation",
	}

	txCmd.AddCommand(
		RegistryDotSamaChain(),
	)

	return txCmd
}

// RegistryDotSamaChain returns the command handler for registry dotsame token info.
func RegistryDotSamaChain() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry",
		Short:   "registry dotsama chain infomation",
		Long:    "registry dotsama chain infomation",
		Args:    cobra.MatchAll(cobra.ExactArgs(4), cobra.OnlyValidArgs),
		Example: fmt.Sprintf("%s tx transfermiddleware registry [ibc_denom] [native_denom] [asset_id] [channel_id]", version.AppName),
		RunE: func(cmd *cobra.Command, args []string) error {
			ibcDenom := args[0]
			nativeDenom := args[1]
			assetID := args[2]
			channelID := args[3]

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			fromAddress := clientCtx.GetFromAddress().String()

			msg := types.NewMsgAddParachainIBCTokenInfo(
				fromAddress,
				ibcDenom,
				nativeDenom,
				assetID,
				channelID,
			)

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}