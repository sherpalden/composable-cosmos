package keeper

import (
	"encoding/json"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/notional-labs/centauri/v3/x/ratelimit/types"
)

type RateLimitedPacketInfo struct {
	ChannelID string
	Denom     string
	Amount    math.Int
	Sender    string
	Receiver  string
}

// Parse the denom from the Send Packet that will be used by the rate limit module
// The denom that the rate limiter will use for a SEND packet depends on whether
// it was a NATIVE token (e.g. ustrd, stuatom, etc.) or NON-NATIVE token (e.g. ibc/...)...
//
// We can identify if the token is native or not by parsing the trace denom from the packet
// If the token is NATIVE, it will not have a prefix (e.g. ustrd),
// and if it is NON-NATIVE, it will have a prefix (e.g. transfer/channel-2/uosmo)
//
// For NATIVE denoms, return as is (e.g. ustrd)
// For NON-NATIVE denoms, take the ibc hash (e.g. hash "transfer/channel-2/usoms" into "ibc/...")
func ParseDenomFromSendPacket(packet transfertypes.FungibleTokenPacketData) (denom string) {
	// Determine the denom by looking at the denom trace path
	denomTrace := transfertypes.ParseDenomTrace(packet.Denom)

	// Native assets will have an empty trace path and can be returned as is
	if denomTrace.Path == "" {
		denom = packet.Denom
	} else {
		// Non-native assets should be hashed
		denom = denomTrace.IBCDenom()
	}

	return denom
}

// Parse the denom from the Recv Packet that will be used by the rate limit module
// The denom that the rate limiter will use for a RECEIVE packet depends on whether it was a source or sink
//
//	Sink:   The token moves forward, to a chain different than its previous hop
//	        The new port and channel are APPENDED to the denom trace.
//	        (e.g. A -> B, B is a sink) (e.g. A -> B -> C, C is a sink)
//
//	Source: The token moves backwards (i.e. revisits the last chain it was sent from)
//	        The port and channel are REMOVED from the denom trace - undoing the last hop.
//	        (e.g. A -> B -> A, A is a source) (e.g. A -> B -> C -> B, B is a source)
//
//	If the chain is acting as a SINK: We add on the Stride port and channel and hash it
//	  Ex1: uosmo sent from Osmosis to Stride
//	       Packet Denom:   uosmo
//	        -> Add Prefix: transfer/channel-X/uosmo
//	        -> Hash:       ibc/...
//
//	  Ex2: ujuno sent from Osmosis to Stride
//	       PacketDenom:    transfer/channel-Y/ujuno  (channel-Y is the Juno <> Osmosis channel)
//	        -> Add Prefix: transfer/channel-X/transfer/channel-Y/ujuno
//	        -> Hash:       ibc/...
//
//	If the chain is acting as a SOURCE: First, remove the prefix. Then if there is still a denom trace, hash it
//	  Ex1: ustrd sent back to Stride from Osmosis
//	       Packet Denom:      transfer/channel-X/ustrd
//	        -> Remove Prefix: ustrd
//	        -> Leave as is:   ustrd
//
//	  Ex2: juno was sent to Stride, then to Osmosis, then back to Stride
//	       Packet Denom:      transfer/channel-X/transfer/channel-Z/ujuno
//	        -> Remove Prefix: transfer/channel-Z/ujuno
//	        -> Hash:          ibc/...
func ParseDenomFromRecvPacket(packet channeltypes.Packet, packetData transfertypes.FungibleTokenPacketData) (denom string) {
	// To determine the denom, first check whether Stride is acting as source
	if transfertypes.ReceiverChainIsSource(packet.GetSourcePort(), packet.GetSourceChannel(), packetData.Denom) {
		// Remove the source prefix (e.g. transfer/channel-X/transfer/channel-Z/ujuno -> transfer/channel-Z/ujuno)
		sourcePrefix := transfertypes.GetDenomPrefix(packet.GetSourcePort(), packet.GetSourceChannel())
		unprefixedDenom := packetData.Denom[len(sourcePrefix):]

		// Native assets will have an empty trace path and can be returned as is
		denomTrace := transfertypes.ParseDenomTrace(unprefixedDenom)
		if denomTrace.Path == "" {
			denom = unprefixedDenom
		} else {
			// Non-native assets should be hashed
			denom = denomTrace.IBCDenom()
		}
	} else {
		// Prefix the destination channel - this will contain the trailing slash (e.g. transfer/channel-X/)
		destinationPrefix := transfertypes.GetDenomPrefix(packet.GetDestPort(), packet.GetDestChannel())
		prefixedDenom := destinationPrefix + packetData.Denom

		// Hash the denom trace
		denomTrace := transfertypes.ParseDenomTrace(prefixedDenom)
		denom = denomTrace.IBCDenom()
	}

	return denom
}

// Parses the sender and channelId and denom for the corresponding RateLimit object, and
// the sender/receiver/transfer amount
//
// The Stride channelID should always be used as the key for the RateLimit object (not the counterparty channelID)
// For a SEND packet, the Stride channelID is the SOURCE channel
// For a RECEIVE packet, the Stride channelID is the DESTINATION channel
//
// The Source and Desination are defined from the perspective of a packet recipient
// Meaning, when a send packet lands on a the host chain, the "Source" will be the Stride Channel,
// and the "Destination" will be the Host Channel
// And, when a receive packet lands on a Stride, the "Source" will be the host zone's channel,
// and the "Destination" will be the Stride Channel
func ParsePacketInfo(packet channeltypes.Packet, direction types.PacketDirection) (RateLimitedPacketInfo, error) {
	var packetData transfertypes.FungibleTokenPacketData
	if err := json.Unmarshal(packet.GetData(), &packetData); err != nil {
		return RateLimitedPacketInfo{}, err
	}

	var channelID, denom string
	if direction == types.PACKET_SEND {
		channelID = packet.GetSourceChannel()
		denom = ParseDenomFromSendPacket(packetData)
	} else {
		channelID = packet.GetDestChannel()
		denom = ParseDenomFromRecvPacket(packet, packetData)
	}

	amount, ok := sdk.NewIntFromString(packetData.Amount)
	if !ok {
		return RateLimitedPacketInfo{},
			errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "Unable to cast packet amount '%s' to sdkmath.Int", packetData.Amount)
	}

	packetInfo := RateLimitedPacketInfo{
		ChannelID: channelID,
		Denom:     denom,
		Amount:    amount,
		Sender:    packetData.Sender,
		Receiver:  packetData.Receiver,
	}

	return packetInfo, nil
}

// Middleware implementation for SendPacket with rate limiting
// Checks whether the rate limit has been exceeded - and if it hasn't, sends the packet
func (k Keeper) SendRateLimitedPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	packetInfo, err := ParsePacketInfo(packet, types.PACKET_SEND)
	if err != nil {
		return err
	}

	// Check if the packet would exceed the outflow rate limit
	updatedFlow, err := k.CheckRateLimitAndUpdateFlow(ctx, types.PACKET_SEND, packetInfo)
	if err != nil {
		return err
	}

	// Store the sequence number of the packet so that if the transfer fails,
	// we can identify if it was sent during this quota and can revert the outflow
	if updatedFlow {
		k.SetPendingSendPacket(ctx, packetInfo.ChannelID, packet.Sequence)
	}

	return nil
}

// Middleware implementation for RecvPacket with rate limiting
// Checks whether the rate limit has been exceeded - and if it hasn't, allows the packet
func (k Keeper) ReceiveRateLimitedPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	packetInfo, err := ParsePacketInfo(packet, types.PACKET_RECV)
	if err != nil {
		return err
	}

	_, err = k.CheckRateLimitAndUpdateFlow(ctx, types.PACKET_RECV, packetInfo)
	return err
}

// Middleware implementation for OnAckPacket with rate limiting
// If the packet failed, we should decrement the Outflow
func (k Keeper) AcknowledgeRateLimitedPacket(ctx sdk.Context, packet channeltypes.Packet, ack []byte) error {
	// Parse the denom, channelId, and amount from the packet
	packetInfo, err := ParsePacketInfo(packet, types.PACKET_SEND)
	if err != nil {
		return err
	}

	var acknowledgement channeltypes.Acknowledgement
	if err := ibctransfertypes.ModuleCdc.UnmarshalJSON(ack, &acknowledgement); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "cannot unmarshal ICS-20 transfer packet acknowledgement: %s", err.Error())
	}

	// The ack can come back as either AcknowledgementResult or AcknowledgementError
	// If it comes back as AcknowledgementResult, the messages are encoded differently depending on the SDK version
	switch response := acknowledgement.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		if len(response.Result) == 0 {
			return errorsmod.Wrapf(channeltypes.ErrInvalidAcknowledgement, "acknowledgement result cannot be empty")
		}

	case *channeltypes.Acknowledgement_Error:
		k.Logger(ctx).Error(fmt.Sprintf("acknowledgement error: %s", response.Error))
		logger.Error()
		return &types.AcknowledgementResponse{Status: types.AckResponseStatus_FAILURE, Error: response.Error}, nil
	default:
		return nil, errorsmod.Wrapf(channeltypes.ErrInvalidAcknowledgement, "unsupported acknowledgement response field type %T", response)
	}
}
