pragma solidity 0.4.24;
pragma experimental ABIEncoderV2;

contract HelloWorld2 {
    event BabyBorn(string name);
    function sayHi () public returns (string) {
        emit BabyBorn("mitzi");
        return 'hi';
    }
    struct ComplexStruct {
        string something;
    }
    function returnStruct () public view returns(ComplexStruct memory complexStruct) {
        complexStruct.something = "sunflower";
    }
}