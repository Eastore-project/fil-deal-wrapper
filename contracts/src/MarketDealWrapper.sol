// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

// Import Statements
import {MarketAPI} from "lib/filecoin-solidity/contracts/v0.8/MarketAPI.sol";
import {CommonTypes} from "lib/filecoin-solidity/contracts/v0.8/types/CommonTypes.sol";
import {MarketTypes} from "lib/filecoin-solidity/contracts/v0.8/types/MarketTypes.sol";
import {AccountTypes} from "lib/filecoin-solidity/contracts/v0.8/types/AccountTypes.sol";
import {AccountCBOR} from "lib/filecoin-solidity/contracts/v0.8/cbor/AccountCbor.sol";
import {MarketCBOR} from "lib/filecoin-solidity/contracts/v0.8/cbor/MarketCbor.sol";
import {BytesCBOR} from "lib/filecoin-solidity/contracts/v0.8/cbor/BytesCbor.sol";
import {BigInts} from "lib/filecoin-solidity/contracts/v0.8/utils/BigInts.sol";
import {CBOR} from "lib/filecoin-solidity/lib/solidity-cborutils/contracts/CBOR.sol";
import {Misc} from "lib/filecoin-solidity/contracts/v0.8/utils/Misc.sol";
import {FilAddresses} from "lib/filecoin-solidity/contracts/v0.8/utils/FilAddresses.sol";
import {Strings} from "lib/openzeppelin-contracts/contracts/utils/Strings.sol";
import {Ownable} from "lib/openzeppelin-contracts/contracts/access/Ownable.sol";
import {IERC20} from "lib/openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";
import {AccountAPI} from "lib/filecoin-solidity/contracts/v0.8/AccountAPI.sol";

// Events
event DealNotify(
    uint64 dealId,
    bytes commP,
    bytes data,
    bytes chainId,
    bytes provider
);
event ReceivedDataCap(string received);
event AddressWhitelisted(address indexed account);
event AddressRemovedFromWhitelist(address indexed account);
event StorageProviderAdded(
    uint64 indexed actorId,
    address indexed ethAddr,
    uint256 pricePerBytePerEpoch
);
event StorageProviderUpdated(
    uint64 indexed actorId,
    address indexed ethAddr,
    uint256 pricePerBytePerEpoch
);
event FundsAdded(address indexed owner, uint256 amount);
event FundsWithdrawn(address indexed owner, uint256 amount);
event SpPaymentCreated(uint64 indexed dealId, uint256 total);
event SpPaymentWithdrawn(
    uint64 indexed dealId,
    address indexed sp,
    uint256 amount
);
event FundsAddedToken(
    address indexed owner,
    address indexed token,
    uint256 amount
);
event FundsWithdrawnToken(
    address indexed owner,
    address indexed token,
    uint256 amount
);
event SpPaymentWithdrawnToken(
    address indexed sp,
    address indexed token,
    uint256 amount
);
event ActorIdWhitelisted(uint64 actorId);
event ActorIdRemovedFromWhitelist(uint64 actorId);

// Errors
error UnauthorizedMarketActor();
error UnauthorizedDataCapActor();
error UnauthorizedMethod();
error InvalidSignature();
error UnauthorizedSender();
error InsufficientBalance();
error ApprovalFailed();
error TransferFailed();
error NoFundsToClaim();
error ContractBalanceTooLow();
error NotStorageProvider();
error InvalidAsciiByte();
error InvalidAsciiHexLength();

// Libraries
using CBOR for CBOR.CBORBuffer;

// Contracts

/**
 * @title MarketDealWrapper
 * @dev A contract to manage Filecoin deals and automate payments to Storage Providers.
 */
contract MarketDealWrapper is Ownable {
    using AccountCBOR for *;
    using MarketCBOR for *;

    // Type Declarations
    struct StorageProvider {
        uint64 actorId;
        address ethAddr;
        IERC20 token;
        uint256 pricePerBytePerEpoch;
    }

    struct DealPayment {
        uint256 pricePerEpoch;
        uint256 withdrawn;
        IERC20 token;
        address sp;
        uint256 startEpoch;
        uint256 endEpoch;
    }

    // State Variables
    uint64 public constant AUTHENTICATE_MESSAGE_METHOD_NUM = 2643134072;
    uint64 public constant DATACAP_RECEIVER_HOOK_METHOD_NUM = 3726118371;
    uint64 public constant MARKET_NOTIFY_DEAL_METHOD_NUM = 4186741094;
    address public constant MARKET_ACTOR_ETH_ADDRESS =
        address(0xff00000000000000000000000000000000000005);
    address public constant DATACAP_ACTOR_ETH_ADDRESS =
        address(0xfF00000000000000000000000000000000000007);

    mapping(bytes => StorageProvider) public storageProviders;
    mapping(uint64 => bool) public isWhitelisted;
    mapping(uint64 => DealPayment) public dealPayments;
    mapping(address => uint256) public ownerDeposits;
    mapping(address => mapping(IERC20 => uint256)) public ownerTokenDeposits;
    mapping(address => uint64[]) public spToDealIds;

    // Modifiers
    /**
     * @dev Ensures that only whitelisted addresses can execute certain functions.
     */
    modifier onlyWhitelisted(uint64 _actorId) {
        if (!isWhitelisted[_actorId]) {
            revert UnauthorizedSender();
        }
        _;
    }

    // Functions

    /**
     * @notice Constructor initializes the contract with the deployer as the owner.
     * @dev Inherits Ownable with msg.sender as the owner.
     */
    constructor() Ownable(msg.sender) {}


    /**
     * @notice Adds an actor ID to the whitelist.
     * @param _actorId The actor ID to be whitelisted.
     */
    function addToWhitelist(uint64 _actorId) external onlyOwner {
        isWhitelisted[_actorId] = true;
        emit ActorIdWhitelisted(_actorId);
    }

    /**
     * @notice Removes an actor ID from the whitelist.
     * @param _actorId The actor ID to be removed from the whitelist.
     */
    function removeFromWhitelist(uint64 _actorId) external onlyOwner {
        isWhitelisted[_actorId] = false;
        emit ActorIdRemovedFromWhitelist(_actorId);
    }

    /**
     * @notice Handles the receipt of DataCap.
     * @param _params The parameters associated with the DataCap.
     */
    function receiveDataCap(bytes memory _params) internal {
        require(
            msg.sender == DATACAP_ACTOR_ETH_ADDRESS,
            "msg.sender needs to be datacap actor f07"
        );
        emit ReceivedDataCap("DataCap Received!");
        // Add get datacap balance API and store DataCap amount
    }

    /**
     * @notice Authenticates messages from the Market Actor. 
     * AuthenticateMessage is the callback from the market actor into the contract
     * as part of PublishStorageDeals. This message holds the deal proposal from the
     * miner, which needs to be validated by the contract in accordance with the
     * deal requests made and the contract's own policies
     * @param params The cbor byte array of AccountTypes.AuthenticateMessageParams containing the deal proposal and signature.
     */
    function authenticateMessage(bytes memory params) internal view {
        require(
            msg.sender == MARKET_ACTOR_ETH_ADDRESS,
            "msg.sender needs to be market actor f05"
        );

        AccountTypes.AuthenticateMessageParams memory amp = params
            .deserializeAuthenticateMessageParams();
        MarketTypes.DealProposal memory proposal = MarketCBOR
            .deserializeDealProposal(amp.message);
        uint64 actorId = uint64(asciiBytesToUint(proposal.label.data));

        // Use AccountAPI to authenticate signature
        int256 exitCode = AccountAPI.authenticateMessage(
            CommonTypes.FilActorId.wrap(actorId),
            amp
        );
        if (exitCode != 0) {
            revert InvalidSignature();
        }

        // Check if actorId is whitelisted
        if (!isWhitelisted[actorId]) {
            revert UnauthorizedSender();
        }
    }

    /**
     * @notice Handles deal notifications from the Market Actor. 
     * This is the callback from the market actor into the contract at the end
     * of PublishStorageDeals. This message holds the previously approved deal proposal
     * and the associated dealID. The dealID is stored as part of the contract state
     * and the completion of this call marks the success of PublishStorageDeals
     * @param params The cbor byte array of MarketDealNotifyParams containing the deal proposal and deal ID.
     */
    function dealNotify(bytes memory params) internal {
        require(
            msg.sender == MARKET_ACTOR_ETH_ADDRESS,
            "msg.sender needs to be market actor f05"
        );

        MarketTypes.MarketDealNotifyParams memory mdnp = MarketCBOR
            .deserializeMarketDealNotifyParams(params);
        MarketTypes.DealProposal memory proposal = MarketCBOR
            .deserializeDealProposal(mdnp.dealProposal);

        int64 duration = CommonTypes.ChainEpoch.unwrap(proposal.end_epoch) -
            CommonTypes.ChainEpoch.unwrap(proposal.start_epoch);

        StorageProvider memory sp = storageProviders[proposal.provider.data];
        // Revert if SP doesn't exist
        if (sp.ethAddr == address(0)) {
            revert UnauthorizedMarketActor();
        }

        // Calculate total payment
        uint256 pricePerEpoch = uint256(proposal.piece_size) *
            sp.pricePerBytePerEpoch;
        uint256 totalPayment = pricePerEpoch * uint256(int256(duration));

        dealPayments[mdnp.dealId] = DealPayment({
            pricePerEpoch: pricePerEpoch,
            withdrawn: 0,
            token: sp.token,
            sp: sp.ethAddr,
            startEpoch: uint256(
                int256(CommonTypes.ChainEpoch.unwrap(proposal.start_epoch))
            ),
            endEpoch: uint256(
                int256(CommonTypes.ChainEpoch.unwrap(proposal.end_epoch))
            )
        });

        // Track deal for the SP
        spToDealIds[sp.ethAddr].push(mdnp.dealId);

        emit SpPaymentCreated(mdnp.dealId, totalPayment);
        emit DealNotify(
            mdnp.dealId,
            proposal.piece_cid.data,
            params,
            proposal.label.data,
            proposal.provider.data
        );
    }

    /**
     * @notice Universal entry point for any EVM-based actor method calls.
     * @param method FRC42 method number for the specific method hook.
     * @param _codec An unused codec param defining input format.
     * @param params The CBOR encoded byte array parameters associated with the method call.
     * @return A tuple containing exit code, codec and bytes return data.
     */
    function handle_filecoin_method(
        uint64 method,
        uint64 _codec,
        bytes memory params
    ) public returns (uint32, uint64, bytes memory) {
        bytes memory ret;
        uint64 codec;

        // Dispatch methods
        if (method == AUTHENTICATE_MESSAGE_METHOD_NUM) {
            authenticateMessage(params);
            // Return CBOR true to indicate successful verification
            CBOR.CBORBuffer memory buf = CBOR.create(1);
            buf.writeBool(true);
            ret = buf.data();
            codec = Misc.CBOR_CODEC;
        } else if (method == MARKET_NOTIFY_DEAL_METHOD_NUM) {
            dealNotify(params);
        } else if (method == DATACAP_RECEIVER_HOOK_METHOD_NUM) {
            receiveDataCap(params);
        } else {
            revert UnauthorizedMethod();
        }

        return (0, codec, ret);
    }

    /**
     * @notice Adds a new Storage Provider to the contract.
     * @param actorId The actor ID of the Storage Provider.
     * @param ethAddr The Ethereum address of the Storage Provider.
     * @param token The ERC20 token used for payments.
     * @param pricePerBytePerEpoch The price per byte per epoch.
     */
    function addStorageProvider(
        uint64 actorId,
        address ethAddr,
        IERC20 token,
        uint256 pricePerBytePerEpoch
    ) external onlyOwner {
        CommonTypes.FilAddress memory filAddr = FilAddresses.fromActorID(
            actorId
        );
        storageProviders[filAddr.data] = StorageProvider(
            actorId,
            ethAddr,
            token,
            pricePerBytePerEpoch
        );
        emit StorageProviderAdded(actorId, ethAddr, pricePerBytePerEpoch);
    }

    /**
     * @notice Updates an existing Storage Provider's details.
     * @param actorId The actor ID of the Storage Provider.
     * @param ethAddr The new Ethereum address of the Storage Provider.
     * @param token The new ERC20 token used for payments.
     * @param pricePerBytePerEpoch The new price per byte per epoch.
     */
    function updateStorageProvider(
        uint64 actorId,
        address ethAddr,
        IERC20 token,
        uint256 pricePerBytePerEpoch
    ) external onlyOwner {
        CommonTypes.FilAddress memory filAddr = FilAddresses.fromActorID(
            actorId
        );
        storageProviders[filAddr.data] = StorageProvider(
            actorId,
            ethAddr,
            token,
            pricePerBytePerEpoch
        );
        emit StorageProviderUpdated(actorId, ethAddr, pricePerBytePerEpoch);
    }

    /**
     * @notice Adds native funds to the contract.
     * @dev Only the owner can add funds.
     */
    function addFunds() external payable onlyOwner {
        ownerDeposits[msg.sender] += msg.value;
        emit FundsAdded(msg.sender, msg.value);
    }

    /**
     * @notice Withdraws native funds from the contract.
     * @param amount The amount to withdraw.
     * @dev Only the owner can withdraw funds.
     */
    function withdrawFunds(uint256 amount) external onlyOwner {
        if (ownerDeposits[msg.sender] < amount) {
            revert InsufficientBalance();
        }
        ownerDeposits[msg.sender] -= amount;
        (bool sent, ) = msg.sender.call{value: amount}("");
        if (!sent) {
            revert TransferFailed();
        }
        emit FundsWithdrawn(msg.sender, amount);
    }

    /**
     * @notice Adds ERC20 tokens to the contract.
     * @param token The ERC20 token to add.
     * @param amount The amount of tokens to add.
     * @dev Only the owner can add ERC20 funds.
     */
    function addFundsERC20(IERC20 token, uint256 amount) external onlyOwner {
        // Transfer tokens from owner to this contract
        bool success = token.transferFrom(msg.sender, address(this), amount);
        if (!success) {
            revert TransferFailed();
        }

        ownerTokenDeposits[msg.sender][token] += amount;
        emit FundsAddedToken(msg.sender, address(token), amount);
    }

    /**
     * @notice Withdraws ERC20 tokens from the contract.
     * @param token The ERC20 token to withdraw.
     * @param amount The amount of tokens to withdraw.
     * @dev Only the owner can withdraw ERC20 funds.
     */
    function withdrawFundsERC20(
        IERC20 token,
        uint256 amount
    ) external onlyOwner {
        if (ownerTokenDeposits[msg.sender][token] < amount) {
            revert InsufficientBalance();
        }
        ownerTokenDeposits[msg.sender][token] -= amount;
        bool success = token.transfer(msg.sender, amount);
        if (!success) {
            revert TransferFailed();
        }
        emit FundsWithdrawnToken(msg.sender, address(token), amount);
    }

    /**
     * @notice Retrieves the current blockchain epoch.
     * @return The current epoch number.
     */
    function getCurrentEpoch() public view returns (uint256) {
        return block.number;
    }

    /**
     * @notice Returns the currently claimable amount for a given deal.
     * @param dealId The dealId to get funds for.
     * @return The amount claimable by the Storage Provider.
     */
    function getSpFundsForDeal(uint64 dealId) public view returns (uint256) {
        (
            int256 exitCode,
            MarketTypes.GetDealActivationReturn memory result
        ) = MarketAPI.getDealActivation(dealId);

        if (
            exitCode != 0 ||
            CommonTypes.ChainEpoch.unwrap(result.terminated) != 0
        ) {
            return 0;
        }
        DealPayment memory dp = dealPayments[dealId];
        uint256 current = getCurrentEpoch();
        if (current < dp.startEpoch) {
            return 0;
        }
        if (current > dp.endEpoch) {
            current = dp.endEpoch;
        }
        uint256 totalVested = dp.pricePerEpoch * (current - dp.startEpoch);
        return totalVested - dp.withdrawn;
    }

    /**
     * @notice Allows the Storage Provider to withdraw funds for a specific deal.
     * @param dealId The dealId to withdraw funds for.
     * @dev Funds are vested as per epoch.
     */
    function withdrawSpFundsForDeal(uint64 dealId) external {
        DealPayment storage s_dp = dealPayments[dealId];
        DealPayment memory dp = s_dp;
        if (msg.sender != dp.sp) {
            revert NotStorageProvider();
        }
        uint256 claimable = getSpFundsForDeal(dealId);
        if (claimable == 0) {
            revert NoFundsToClaim();
        }
        s_dp.withdrawn += claimable;

        if (address(dp.token) == address(0)) {
            if (address(this).balance < claimable) {
                revert ContractBalanceTooLow();
            }
            (bool sent, ) = msg.sender.call{value: claimable}("");
            if (!sent) {
                revert TransferFailed();
            }
        } else {
            bool success = dp.token.transfer(msg.sender, claimable);
            if (!success) {
                revert TransferFailed();
            }
        }

        emit SpPaymentWithdrawn(dealId, msg.sender, claimable);
    }

    /**
     * @notice Allows the Storage Provider to withdraw all funds by ERC20 token.
     * @param token The ERC20 token to withdraw.
     */
    function withdrawSpFundsByToken(IERC20 token) external {
        uint64[] memory deals = spToDealIds[msg.sender];
        uint256 totalClaimable;
        uint256 dealCount = deals.length;

        for (uint256 i = 0; i < dealCount; i++) {
            uint64 dealId = deals[i];
            DealPayment memory dp = dealPayments[dealId];
            if (dp.sp != msg.sender) {
                continue;
            }
            if (address(dp.token) != address(token)) {
                continue;
            }
            DealPayment storage s_dp = dealPayments[dealId];
            uint256 claimable = getSpFundsForDeal(dealId);
            if (claimable > 0) {
                s_dp.withdrawn += claimable;
                totalClaimable += claimable;
            }
        }

        if (totalClaimable == 0) {
            revert NoFundsToClaim();
        }

        if (address(token) == address(0)) {
            // Native coin
            if (address(this).balance < totalClaimable) {
                revert ContractBalanceTooLow();
            }
            (bool sent, ) = msg.sender.call{value: totalClaimable}("");
            if (!sent) {
                revert TransferFailed();
            }
        } else {
            bool success = token.transfer(msg.sender, totalClaimable);
            if (!success) {
                revert TransferFailed();
            }
        }

        // Emit one aggregated event
        emit SpPaymentWithdrawnToken(
            msg.sender,
            address(token),
            totalClaimable
        );
    }

    /**
     * @notice Allows the Storage Provider to withdraw partial funds for a terminated deal.
     * @param dealId The dealId of the terminated deal.
     */
    function withdrawSpFundsForTerminatedDeal(uint64 dealId) external {
        (
            int256 exitCode,
            MarketTypes.GetDealActivationReturn memory result
        ) = MarketAPI.getDealActivation(dealId);
        if (
            exitCode != 0 ||
            CommonTypes.ChainEpoch.unwrap(result.terminated) == 0
        ) {
            revert("Deal not terminated or not found");
        }

        DealPayment storage s_dp = dealPayments[dealId];
        DealPayment memory dp = s_dp;
        if (msg.sender != dp.sp) {
            revert NotStorageProvider();
        }

        dp.endEpoch = uint256(
            int256(CommonTypes.ChainEpoch.unwrap(result.terminated))
        );
        s_dp.endEpoch = dp.endEpoch;

        if (msg.sender != dp.sp) {
            revert NotStorageProvider();
        }

        uint256 current = getCurrentEpoch();
        if (current > dp.endEpoch) {
            current = dp.endEpoch;
        }
        uint256 totalVested = dp.pricePerEpoch * (current - dp.startEpoch);

        uint256 claimable = totalVested - dp.withdrawn;
        if (claimable == 0) {
            revert NoFundsToClaim();
        }
        s_dp.withdrawn += claimable;

        if (address(dp.token) == address(0)) {
            if (address(this).balance < claimable) {
                revert ContractBalanceTooLow();
            }
            (bool sent, ) = msg.sender.call{value: claimable}("");
            if (!sent) {
                revert TransferFailed();
            }
        } else {
            bool success = dp.token.transfer(msg.sender, claimable);
            if (!success) {
                revert TransferFailed();
            }
        }

        emit SpPaymentWithdrawn(dealId, msg.sender, claimable);
    }

    /**
     * @notice Retrieves deal IDs associated with a given miner ID.
     * @param minerId The miner ID.
     * @return An array of deal IDs associated with the miner.
     */
    function getDealsFromMinerId(
        uint64 minerId
    ) public view returns (uint64[] memory) {
        // Get FilAddress from minerId
        CommonTypes.FilAddress memory filAddr = FilAddresses.fromActorID(
            minerId
        );

        // Get the StorageProvider using filAddr.data
        StorageProvider memory sp = storageProviders[filAddr.data];

        // Retrieve deal IDs from spToDealIds using sp.ethAddr
        return spToDealIds[sp.ethAddr];
    }

    /**
     * @notice Retrieves the currently claimable SP funds for a specific ERC20 token and actor ID.
     * @param token The ERC20 token.
     * @param actorId The actor ID of the Storage Provider.
     * @return The total claimable funds.
     */
    function getTokenFundsForSp(
        IERC20 token,
        uint64 actorId
    ) public view returns (uint256) {
        address spAddr = getSpFromId(actorId).ethAddr;
        uint64[] memory deals = spToDealIds[spAddr];
        uint256 totalClaimable;
        uint256 dealCount = deals.length;

        for (uint256 i = 0; i < dealCount; i++) {
            uint64 dealId = deals[i];
            DealPayment memory dp = dealPayments[dealId];
            if (dp.sp != spAddr) {
                continue;
            }
            if (address(dp.token) != address(token)) {
                continue;
            }
            uint256 claimable = getSpFundsForDeal(dealId);
            totalClaimable += claimable;
        }

        return totalClaimable;
    }

    /**
     * @notice Retrieves the Storage Provider details from an actor ID.
     * @param actorId The actor ID of the Storage Provider.
     * @return The StorageProvider struct.
     */
    function getSpFromId(
        uint64 actorId
    ) public view returns (StorageProvider memory) {
        CommonTypes.FilAddress memory filAddr = FilAddresses.fromActorID(
            actorId
        );
        return storageProviders[filAddr.data];
    }

    /**
     * @notice Converts an address to its hexadecimal string representation.
     * @param _addr The address to convert.
     * @return The hexadecimal string representation of the address.
     */
    function addressToHexString(
        address _addr
    ) internal pure returns (string memory) {
        return Strings.toHexString(uint256(uint160(_addr)), 20);
    }

    /**
     * @notice Converts ASCII bytes to a uint256.
     * @param asciiBytes The ASCII bytes to convert.
     * @return The resulting uint256.
     */
    function asciiBytesToUint(
        bytes memory asciiBytes
    ) public pure returns (uint256) {
        uint256 result = 0;
        for (uint256 i = 0; i < asciiBytes.length; i++) {
            uint256 digit = uint256(uint8(asciiBytes[i])) - 48; // Convert ASCII to digit
            if (digit > 9) {
                revert InvalidAsciiByte();
            }
            result = result * 10 + digit;
        }
        return result;
    }

    /**
     * @notice Converts ASCII hexadecimal bytes to bytes.
     * @param asciiHex The ASCII hexadecimal bytes to convert.
     * @return The resulting bytes.
     */
    function convertAsciiHexToBytes(
        bytes memory asciiHex
    ) public pure returns (bytes memory) {
        if (asciiHex.length % 2 != 0) {
            revert InvalidAsciiHexLength();
        }

        bytes memory result = new bytes(asciiHex.length / 2);
        for (uint256 i = 0; i < asciiHex.length / 2; i++) {
            result[i] = byteFromHexChar(asciiHex[2 * i], asciiHex[2 * i + 1]);
        }

        return result;
    }

    /**
     * @notice Converts two hexadecimal characters to a single byte.
     * @param char1 The first hexadecimal character.
     * @param char2 The second hexadecimal character.
     * @return The resulting byte.
     */
    function byteFromHexChar(
        bytes1 char1,
        bytes1 char2
    ) internal pure returns (bytes1) {
        uint8 nibble1 = uint8(char1) - (uint8(char1) < 58 ? 48 : 87);
        uint8 nibble2 = uint8(char2) - (uint8(char2) < 58 ? 48 : 87);
        return bytes1(nibble1 * 16 + nibble2);
    }

    /**
     * @notice Recovers the signer address from a hash and signature.
     * @dev Could be replaced by openzeppelin ECDSA library.
     * @param hash The hash that was signed.
     * @param signature The signature bytes.
     * @return The recovered address.
     */
    function recovers(
        bytes32 hash,
        bytes memory signature
    ) public pure returns (address) {
        bytes32 r;
        bytes32 s;
        uint8 v;

        // Check the signature length
        if (signature.length != 65) {
            return address(0);
        }

        // Divide the signature into r, s, and v variables
        assembly {
            r := mload(add(signature, 32))
            s := mload(add(signature, 64))
            v := byte(0, mload(add(signature, 96)))
        }

        // Adjust the version of the signature
        if (v < 27) {
            v += 27;
        }

        // Return address(0) if the version is incorrect
        if (v != 27 && v != 28) {
            return address(0);
        } else {
            return ecrecover(hash, v, r, s);
        }
    }

    /**
     * @notice Fallback function to receive Ether.
     */
    receive() external payable {}

    /**
     * @notice Fallback function.
     */
    fallback() external payable {}
}