// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.27;

contract Panicing {
    address public exchangeContract;
    event UserRevert(address indexed sender, string reason);

    constructor(address _exchangeContract) {
        exchangeContract = _exchangeContract;
    }

    // panicDurningCall is invoking a precompile with malformed input, should revert
    function panicDurningCall() external {
        (bool success, bytes memory error) = exchangeContract.call("");

        if (success) {
            // If call succeeds, it's a bug in the test
            revert("call should have failed");
        } else {
            revert(string(abi.encodePacked("panic during call", error)));
        }
    }

    // userRevert will always revert, useful for testing reverts with a reason
    // via events.
    function userRevert(string memory reason) public {
        emit UserRevert(msg.sender, reason);
        revert("user requested a revert; see event for details");
    }
}
