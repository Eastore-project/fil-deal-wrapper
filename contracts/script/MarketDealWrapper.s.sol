// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.27;

import {Script, console} from "forge-std/Script.sol";
import {MarketDealWrapper} from "../src/MarketDealWrapper.sol";

contract MarketDealWrapperScript is Script {
    MarketDealWrapper public marketDealWrapper;

    function setUp() public {}

    function run() public {
        vm.startBroadcast();

        marketDealWrapper = new MarketDealWrapper();

        vm.stopBroadcast();
    }
}
