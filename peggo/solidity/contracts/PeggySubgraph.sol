// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import "./Peggy.sol";

// Legacy events and functions for indexing in subgraph

contract PeggySubgraph is Peggy {
    using SafeERC20 for IERC20;

    event SendToCosmosEvent(
        address indexed _tokenContract,
        address indexed _sender,
        bytes32 indexed _destination,
        uint256 _amount,
        uint256 _eventNonce
    );

    function sendToCosmos(
        address _tokenContract,
        bytes32 _destination,
        uint256 _amount
    ) external whenNotPaused nonReentrant {
        IERC20(_tokenContract).safeTransferFrom(
            msg.sender,
            address(this),
            _amount
        );
        state_lastEventNonce = state_lastEventNonce + 1;
        emit SendToCosmosEvent(
            _tokenContract,
            msg.sender,
            _destination,
            _amount,
            state_lastEventNonce
        );
    }
}
