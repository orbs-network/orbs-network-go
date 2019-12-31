#!/usr/bin/env node

/*
This script updates a local Boyar config.json file to disable an existing PR chain

Usage: 

$ testnet-disable-chain.js <ChainID>

Examples:

$ testnet-disable-chain.js 1104832

*/

const path = require('path');
const fs = require('fs');

const targetChainId = process.argv[2];
const configFilePath = path.join(process.cwd(), 'config.json');

if (!targetChainId) {
    console.log('No chainId given!');
    process.exit(1);
}

const configuration = require(configFilePath);

const chainIndex = configuration.chains.findIndex(chain => chain.Id === parseInt(targetChainId));

if (chainIndex !== -1) {
    console.log('Chain found, Disabled in config..');
    configuration.chains[chainIndex].Disabled = true;
} else {
    console.log('Chain was not found within the configuration!');
    process.exit(2);
}

fs.writeFileSync(configFilePath, JSON.stringify(configuration, 2, 2));

process.exit(0);