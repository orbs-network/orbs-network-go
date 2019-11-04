package main

import (
	"github.com/orbs-network/contract-external-libraries-go/v1/list"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
)

var PUBLIC = sdk.Export(get, add)
var SYSTEM = sdk.Export(_init)

func _init() {

}

func get(index uint64) string {
	list := getList()
	return list.Get(index).(string)
}

func add(item string) uint64 {
	return getList().Append(item)
}

func getList() list.List {
	return list.NewAppendOnlyList("items", list.StringSerializer, list.StringDeserializer)
}
