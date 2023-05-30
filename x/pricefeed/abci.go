package pricefeed

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/mage-coven/fury/x/pricefeed/keeper"
	"github.com/mage-coven/fury/x/pricefeed/types"
)

// EndBlocker updates the current pricefeed
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	// Update the current price of each asset.
	for _, market := range k.GetMarkets(ctx) {
		if !market.Active {
			continue
		}

		err := k.SetCurrentPrices(ctx, market.MarketID)
		if err != nil && !errors.Is(err, types.ErrNoValidPrice) {
			panic(err)
		}
	}
}
