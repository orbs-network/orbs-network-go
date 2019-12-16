pragma solidity 0.4.26;

contract FakeVoting  {
    uint internal delegationCounter;

    event Delegate(
        address indexed delegator,
        address indexed to,
        uint delegationCounter
    );

    function delegate(address to) external {
        address sender = msg.sender;
        require(to != address(0), "must delegate to non 0");
        require(sender != to , "cant delegate to yourself");

        delegationCounter++;

        emit Delegate(sender, to, delegationCounter);
    }
}
