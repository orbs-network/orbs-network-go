pragma solidity ^0.5.0;
contract Logger {
    event Log(int32 count);

    function log(int32 count) public {

        emit Log(count);
    }
}
