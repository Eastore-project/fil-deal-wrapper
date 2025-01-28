// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import "forge-std/Test.sol";
import "../src/MarketDealWrapper.sol";
import "lib/openzeppelin-contracts/contracts/token/ERC20/ERC20.sol";

// Mock ERC20 Token for testing
contract MockERC20 is ERC20 {
    constructor(
        string memory name_,
        string memory symbol_
    ) ERC20(name_, symbol_) {}

    function mint(address to, uint256 amount) external {
        _mint(to, amount);
    }
}

contract MarketDealWrapperTest is Test {
    MarketDealWrapper wrapper;
    MockERC20 mockToken;
    address owner = address(1);
    address user = address(2);
    address sp = address(3);
    uint64 constant userActorId = 1010;

    function setUp() public {
        // Deploy Mock ERC20 Token
        mockToken = new MockERC20("Mock Token", "MTK");
        // Deploy MarketDealWrapper with owner
        vm.startPrank(owner);
        wrapper = new MarketDealWrapper();
        vm.stopPrank();

        // Mint tokens to owner
        mockToken.mint(owner, 1000 ether);

        // Fund the owner with ether for testing
        vm.deal(owner, 10 ether);
    }

    // Test Ownership
    function testOwner() public view {
        assertEq(wrapper.owner(), owner, "Owner should be deployer");
    }

    function testOnlyOwnerCanAddToWhitelist() public {
        vm.prank(user);
        vm.expectRevert();
        wrapper.addToWhitelist(userActorId); // changed from user
    }

    function testOwnerCanAddToWhitelist() public {
        vm.prank(owner);
        wrapper.addToWhitelist(userActorId); // changed from user
        assertTrue(
            wrapper.isWhitelisted(userActorId),
            "Actor ID should be whitelisted"
        ); // changed from user
    }

    function testOnlyOwnerCanRemoveFromWhitelist() public {
        // First, add to whitelist
        vm.prank(owner);
        wrapper.addToWhitelist(userActorId); // changed from user

        vm.prank(user);
        vm.expectRevert();
        wrapper.removeFromWhitelist(userActorId); // changed from user
    }

    function testOwnerCanRemoveFromWhitelist() public {
        // First, add to whitelist
        vm.prank(owner);
        wrapper.addToWhitelist(userActorId); // changed from user

        // Remove from whitelist
        vm.prank(owner);
        wrapper.removeFromWhitelist(userActorId); // changed from user
        assertFalse(
            wrapper.isWhitelisted(userActorId), // changed from user
            "Actor ID should be removed from whitelist" // changed from user
        );
    }

    // Test Adding and Withdrawing Native Funds
    function testAddFunds() public {
        vm.prank(owner);
        wrapper.addFunds{value: 1 ether}();
        assertEq(
            wrapper.ownerDeposits(owner),
            1 ether,
            "Owner deposit should be 1 ether"
        );
    }

    function testWithdrawFunds() public {
        // Add funds first
        vm.prank(owner);
        wrapper.addFunds{value: 2 ether}();

        // Withdraw funds
        vm.prank(owner);
        wrapper.withdrawFunds(1 ether);
        assertEq(
            wrapper.ownerDeposits(owner),
            1 ether,
            "Owner deposit should be 1 ether after withdrawal"
        );
    }

    function testWithdrawFundsInsufficientBalance() public {
        vm.prank(owner);
        vm.expectRevert(bytes4(keccak256("InsufficientBalance()")));
        wrapper.withdrawFunds(1 ether);
    }

    // Test Adding and Withdrawing ERC20 Funds
    function testAddFundsERC20() public {
        // Approve tokens
        vm.prank(owner);
        mockToken.approve(address(wrapper), 500 ether);

        // Add ERC20 funds
        vm.prank(owner);
        wrapper.addFundsERC20(mockToken, 500 ether);
        assertEq(
            wrapper.ownerTokenDeposits(owner, mockToken),
            500 ether,
            "Owner ERC20 deposit should be 500 ether"
        );
    }

    function testWithdrawFundsERC20() public {
        // Approve and add funds
        vm.prank(owner);
        mockToken.approve(address(wrapper), 500 ether);
        vm.prank(owner);
        wrapper.addFundsERC20(mockToken, 500 ether);

        // Withdraw ERC20 funds
        vm.prank(owner);
        wrapper.withdrawFundsERC20(mockToken, 200 ether);
        assertEq(
            wrapper.ownerTokenDeposits(owner, mockToken),
            300 ether,
            "Owner ERC20 deposit should be 300 ether after withdrawal"
        );
    }

    function testWithdrawFundsERC20InsufficientBalance() public {
        // Approve and add funds
        vm.prank(owner);
        mockToken.approve(address(wrapper), 100 ether);
        vm.prank(owner);
        wrapper.addFundsERC20(mockToken, 100 ether);

        // Attempt to withdraw more than balance
        vm.prank(owner);
        vm.expectRevert(bytes4(keccak256("InsufficientBalance()")));
        wrapper.withdrawFundsERC20(mockToken, 200 ether);
    }

    // // Test Deal Notification
    // function testDealNotify() public {
    //     // Assuming proper setup of dealNotify parameters
    //     bytes memory params = ""; // Populate with valid CBOR-encoded data as per contract's expectation

    //     // Mock msg.sender as MARKET_ACTOR_ETH_ADDRESS
    //     vm.prank(address(wrapper.MARKET_ACTOR_ETH_ADDRESS()));
    //     // Note: The actual parameters should be valid according to the contract's deserialize functions
    //     // For simplicity, this test assumes that the function handles empty params without reverting
    //     // In real scenarios, proper CBOR-encoded data should be provided

    //     // vm.expectEmit(true, true, true, true);
    //     // emit DealNotify(...); // Define expected emit parameters

    //     wrapper.dealNotify(params);

    //     // Add assertions based on the expected state changes
    // }

    // Additional tests can be added for authenticateMessage, withdrawMinerFundsForDeal, etc.
}
