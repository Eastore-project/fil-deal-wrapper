// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Import OpenZeppelin's ERC20 implementation
import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/**
 * @title SimpleERC20
 * @dev ERC20 Token that allows anyone to mint a fixed amount of tokens.
 *      Upon deployment, a predefined number of tokens are transferred to the deployer.
 */
contract SimpleERC20 is ERC20 {
    // Fixed amount of tokens to mint per call (e.g., 100 tokens)
    uint256 public constant MINT_AMOUNT = 100 * 10 ** 18;

    /**
     * @dev Constructor that gives msg.sender an initial supply of tokens.
     */
    constructor() ERC20("SimpleERC20", "SIM") {
        // Mint the initial supply to the deployer
        _mint(msg.sender, MINT_AMOUNT);
    }

    /**
     * @dev Allows anyone to mint a fixed amount of tokens to their address.
     */
    function mint() external {
        _mint(msg.sender, MINT_AMOUNT);
    }
}
