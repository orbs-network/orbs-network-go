package deployments_systemcontract

import "github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"

var PUBLIC = sdk.Export(getInfo, getCode, deployService, lockNativeDeployment, unlockNativeDeployment)
