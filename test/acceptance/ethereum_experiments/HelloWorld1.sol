pragma solidity 0.4.24;
pragma experimental ABIEncoderV2;

import "./HelloWorld2.sol";

contract HelloWorld1 {
    event BabyBorn(string name);
    function sayHi (address helloWorld2Address) public returns (string) {
        HelloWorld2 helloWorld2Instance = HelloWorld2(helloWorld2Address);
        HelloWorld2.ComplexStruct memory returnedStruct = helloWorld2Instance.returnStruct();
        emit BabyBorn(returnedStruct.something);
        return 'hi';
    }
}