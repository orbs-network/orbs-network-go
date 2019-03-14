pragma solidity ^0.4.24;

interface ISubscriptionChecker {
    /// @param _id the virtual chain id to check subscription for
    /// @return profile - the subscribed plan, e.g. 'gold', 'silver', etc
    function getSubscriptionData(bytes32 _id) external view returns (bytes32 id, string profile, uint256 startTime, uint256 tokens);
}

contract FakeSubscriptionChecker is ISubscriptionChecker {
    function getSubscriptionData(bytes32 _id) public view returns (bytes32 id, string profile, uint256 startTime, uint256 tokens) {
        uint256 intId = uint256(_id);
        if (intId == 42) {
            return (_id, "B4", 0, 6600);
        } else if (intId == 17) {
            return (_id, "B2", 0, 1000); // underfunded
        }
    }
}
