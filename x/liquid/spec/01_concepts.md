<!--
order: 1
-->

# Concepts

This module is responsible for the minting and burning of liquid staking receipt tokens, collectively referred to as `bfury`. Delegated fury can be converted to delegator-specific `bfury`. Ie, 100 FURY delegated to validator `furyvaloper123` can be converted to 100 `bfury-furyvaloper123`. Similarly, 100 `bfury-furyvaloper123` can be converted back to a delegation of 100 FURY to  `furyvaloper123`. In this design, all validators can permissionlessly participate in liquid staking while users retain the delegator specific slashing risk and voting rights of their original validator. Note that because each `bfury` denom is validator specific, this module does not specify a fungibility mechanism for `bfury` denoms. 