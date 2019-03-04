pragma solidity ^0.5.0;
contract EmitAddressArrayEvent {
    event Vote(address indexed voter, address[] nodeslist, uint vote_counter);

    function fire(address[] memory addresses) public {
        emit Vote(msg.sender, addresses, 42);
    }
}
