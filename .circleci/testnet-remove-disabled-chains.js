#!/usr/bin/env node

/*
This script updates a local Boyar config.json file to remove all chains which were marked for deletion

Usage: 

$ testnet-remove-disabled-chains.js

*/

const path = require('path');
const fs = require('fs');

const configFilePath = path.join(process.cwd(), 'config.json');
const configuration = require(configFilePath);

configuration.chains = configuration.chains.filter(chain => !chain.Disabled);

fs.writeFileSync(configFilePath, JSON.stringify(configuration, 2, 2));

process.exit(0);