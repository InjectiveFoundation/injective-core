// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.27;

contract Counter {
    uint256 public number;

    event ValueSet(address indexed sender, uint256 newValue);
    event UserRevert(address indexed sender, string reason);

    function setNumber(uint256 newNumber) public {
        number = newNumber;
        emit ValueSet(msg.sender, number);
    }

    function increment() public {
        number++;
        emit ValueSet(msg.sender, number);
    }

    function userRevert(string memory reason) public {
        emit UserRevert(msg.sender, reason);
        revert("user requested a revert; see event for details");
    }
}
