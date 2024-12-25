// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

import {MarketAPI} from "lib/filecoin-solidity/contracts/v0.8/MarketAPI.sol";
import {CommonTypes} from "lib/filecoin-solidity/contracts/v0.8/types/CommonTypes.sol";
import {MarketTypes} from "lib/filecoin-solidity/contracts/v0.8/types/MarketTypes.sol";
import {AccountTypes} from "lib/filecoin-solidity/contracts/v0.8/types/AccountTypes.sol";
import {CommonTypes} from "lib/filecoin-solidity/contracts/v0.8/types/CommonTypes.sol";
import {AccountCBOR} from "lib/filecoin-solidity/contracts/v0.8/cbor/AccountCbor.sol";
import {MarketCBOR} from "lib/filecoin-solidity/contracts/v0.8/cbor/MarketCbor.sol";
import {BytesCBOR} from "lib/filecoin-solidity/contracts/v0.8/cbor/BytesCbor.sol";
import {BigInts} from "lib/filecoin-solidity/contracts/v0.8/utils/BigInts.sol";
import {CBOR} from "lib/filecoin-solidity/lib/solidity-cborutils/contracts/CBOR.sol";
import {Misc} from "lib/filecoin-solidity/contracts/v0.8/utils/Misc.sol";
import {FilAddresses} from "lib/filecoin-solidity/contracts/v0.8/utils/FilAddresses.sol";
import {Strings} from "lib/openzeppelin-contracts/contracts/utils/Strings.sol";
import {BLAKE2b} from "./blake2lib.sol";
import {Ownable} from "lib/openzeppelin-contracts/contracts/access/Ownable.sol";
import {IERC20} from "lib/openzeppelin-contracts/contracts/token/ERC20/IERC20.sol";

// Add custom errors
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

using CBOR for CBOR.CBORBuffer;

contract MarketDealWrapper is Ownable(msg.sender) {
    using AccountCBOR for *;
    using MarketCBOR for *;

    uint64 public constant AUTHENTICATE_MESSAGE_METHOD_NUM = 2643134072;
    uint64 public constant DATACAP_RECEIVER_HOOK_METHOD_NUM = 3726118371;
    uint64 public constant MARKET_NOTIFY_DEAL_METHOD_NUM = 4186741094;
    address public constant MARKET_ACTOR_ETH_ADDRESS =
        address(0xff00000000000000000000000000000000000005);
    address public constant DATACAP_ACTOR_ETH_ADDRESS =
        address(0xfF00000000000000000000000000000000000007);

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

    mapping(bytes => StorageProvider) public storageProviders;
    mapping(address => bool) public isWhitelisted;
    mapping(uint64 => DealPayment) public dealPayments;
    mapping(address => uint256) public ownerDeposits;
    mapping(address => mapping(IERC20 => uint256)) public ownerTokenDeposits;
    mapping(address => uint64[]) public spToDealIds;

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

    function addToWhitelist(address _address) external onlyOwner {
        isWhitelisted[_address] = true;
        emit AddressWhitelisted(_address);
    }

    function removeFromWhitelist(address _address) external onlyOwner {
        isWhitelisted[_address] = false;
        emit AddressRemovedFromWhitelist(_address);
    }

    function receiveDataCap(bytes memory) internal {
        require(
            msg.sender == DATACAP_ACTOR_ETH_ADDRESS,
            "msg.sender needs to be datacap actor f07"
        );
        emit ReceivedDataCap("DataCap Received!");
        // Add get datacap balance api and store datacap amount
    }

    // authenticateMessage is the callback from the market actor into the contract
    // as part of PublishStorageDeals. This message holds the deal proposal from the
    // miner, which needs to be validated by the contract in accordance with the
    // deal requests made and the contract's own policies
    // @params - cbor byte array of AccountTypes.AuthenticateMessageParams
    function authenticateMessage(bytes memory params) internal view {
        require(
            msg.sender == MARKET_ACTOR_ETH_ADDRESS,
            "msg.sender needs to be market actor f05"
        );

        AccountTypes.AuthenticateMessageParams memory amp = params
            .deserializeAuthenticateMessageParams();
        MarketTypes.DealProposal memory proposal = MarketCBOR
            .deserializeDealProposal(amp.message);
        bytes memory encodedData = convertAsciiHexToBytes(proposal.label.data);
        address filAddress = abi.decode(encodedData, (address));
        address recovered = recovers(
            bytes32(BLAKE2b.hash(amp.message, "", "", "", 32)),
            amp.signature
        );
        if (recovered != filAddress) {
            revert InvalidSignature();
        }

        if (!isWhitelisted[recovered]) {
            revert UnauthorizedSender();
        }
    }

    // dealNotify is the callback from the market actor into the contract at the end
    // of PublishStorageDeals. This message holds the previously approved deal proposal
    // and the associated dealID. The dealID is stored as part of the contract state
    // and the completion of this call marks the success of PublishStorageDeals
    // @params - cbor byte array of MarketDealNotifyParams
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
        // revert if SP doesn't exist
        if (sp.ethAddr == address(0)) {
            revert UnauthorizedMarketActor();
        }

        // Naive computation of total payment for demonstration:
        // total = piece_size * sp.pricePerBytePerEpoch * duration
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

        // track deal for the SP
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

    // handle_filecoin_method is the universal entry point for any evm based
    // actor for a call coming from a builtin filecoin actor
    // @method - FRC42 method number for the specific method hook
    // @params - CBOR encoded byte array params
    function handle_filecoin_method(
        uint64 method,
        uint64,
        bytes memory params
    ) public returns (uint32, uint64, bytes memory) {
        bytes memory ret;
        uint64 codec;
        // dispatch methods
        if (method == AUTHENTICATE_MESSAGE_METHOD_NUM) {
            authenticateMessage(params);
            // If we haven't reverted, we should return a CBOR true to indicate that verification passed.
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

    function addFunds() external payable onlyOwner {
        ownerDeposits[msg.sender] += msg.value;
        emit FundsAdded(msg.sender, msg.value);
    }

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

    function addFundsERC20(IERC20 token, uint256 amount) external onlyOwner {
        // Transfer tokens from owner to this contract
        bool success = token.transferFrom(msg.sender, address(this), amount);
        if (!success) {
            revert TransferFailed();
        }

        ownerTokenDeposits[msg.sender][token] += amount;
        emit FundsAddedToken(msg.sender, address(token), amount);
    }

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

    function getCurrentEpoch() public view returns (uint256) {
        return block.number;
    }

    // Returns the currently claimable amount for a given deal (does not update state)
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

    // Renamed existing function
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

        // emit one aggregated event
        emit SpPaymentWithdrawnToken(
            msg.sender,
            address(token),
            totalClaimable
        );
    }

    // Would be called by SP to get partial payment for a terminated deal
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
        // approximate epoch
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

    // Retrieves deal IDs associated with a given miner ID
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

    function getSpFromId(
        uint64 actorId
    ) public view returns (StorageProvider memory) {
        CommonTypes.FilAddress memory filAddr = FilAddresses.fromActorID(
            actorId
        );
        return storageProviders[filAddr.data];
    }

    function addressToHexString(
        address _addr
    ) internal pure returns (string memory) {
        return Strings.toHexString(uint256(uint160(_addr)), 20);
    }

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

    function byteFromHexChar(
        bytes1 char1,
        bytes1 char2
    ) internal pure returns (bytes1) {
        uint8 nibble1 = uint8(char1) - (uint8(char1) < 58 ? 48 : 87);
        uint8 nibble2 = uint8(char2) - (uint8(char2) < 58 ? 48 : 87);
        return bytes1(nibble1 * 16 + nibble2);
    }

    function recovers(
        bytes32 hash,
        bytes memory signature
    ) public pure returns (address) {
        bytes32 r;
        bytes32 s;
        uint8 v;

        // Check the signature length
        if (signature.length != 65) {
            return (address(0));
        }

        // Divide the signature in r, s and v variables
        assembly {
            r := mload(add(signature, 32))
            s := mload(add(signature, 64))
            v := byte(0, mload(add(signature, 96)))
        }

        // Version of signature should be 27 or 28, but 0 and 1 are also possible versions
        if (v < 27) {
            v += 27;
        }
        // address check = ECDSA.recover(hash, signature);
        // If the version is correct return the signer address
        if (v != 27 && v != 28) {
            return (address(0));
        } else {
            return ecrecover(hash, v, r, s);
        }
    }
}
