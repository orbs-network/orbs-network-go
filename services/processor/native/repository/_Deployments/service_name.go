package deployments_systemcontract

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"regexp"
	"strings"
)

func _validateServiceName(serviceName string) {
	if IsImplicitlyDeployed(serviceName) {
		panic("a contract with this name exists")
	}
	if matched, err := regexp.MatchString("^[a-zA-Z0-9]+$", serviceName); err != nil || !matched {
		panic("contract name must be non empty and contain alpha-numeric characters only")
	}
	if _isServiceNameUsed(serviceName) {
		panic("a contract with same name (case insensitive) already exists")
	}
}

func _isServiceNameUsed(serviceName string) bool {
	numOfServices := _getNumberOfServices()
	serviceName = strings.ToLower(serviceName)
	for i := 0; i < numOfServices; i++ {
		if serviceName == _getServiceAtIndex(i) {
			return true
		}
	}
	return false
}

func _addServiceName(serviceName string) {
	numOfServices := _getNumberOfServices()
	_setServiceAtIndex(numOfServices, serviceName)
	numOfServices++
	_setNumberOfServices(numOfServices)
}

func _formatNumberOfDeployedServices() []byte {
	return []byte("Service_Count")
}

func _getNumberOfServices() int {
	return int(state.ReadUint32(_formatNumberOfDeployedServices()))
}

func _setNumberOfServices(numberOfServices int) {
	state.WriteUint32(_formatNumberOfDeployedServices(), uint32(numberOfServices))
}

func _formatServiceIterator(num int) []byte {
	return []byte(fmt.Sprintf("Service_At_%d", num))
}

func _getServiceAtIndex(index int) string {
	return state.ReadString(_formatServiceIterator(index))
}

func _setServiceAtIndex(index int, serviceName string) {
	state.WriteString(_formatServiceIterator(index), strings.ToLower(serviceName))
}
