#!/usr/bin/env node

/*
This script updates a local Boyar config.json file to provision a new chain
for E2E purposes on an isolated blockchain network

Usage: 

$ testnet-deploy-new-chain-for-pr.js <GitHub PR Link>

Examples:

$ testnet-deploy-new-chain-for-pr.js https://github.com/orbs-network/orbs-network-go/pull/1184

*/

const path = require('path');
const fs = require('fs');

const {
    newChainConfiguration,
    getBoyarChainConfigurationById,
    updateChainConfiguration,
    getPrChainNumber,
    newVacantTCPPort,
} = require('./boyar-lib');

const githubPRLink = process.argv[2];
const targetTag = process.argv[3];
const configFilePath = path.join(process.cwd(), 'config.json');

if (!githubPRLink) {
    console.log('No GitHub PR link supplied!');
    process.exit(1);
}

if (!targetTag) {
    console.log('No version hash!');
    process.exit(1);
}

const prLinkParts = githubPRLink.split('/');
const prNumber = parseInt(prLinkParts[prLinkParts.length - 1]);
const chainNumber = getPrChainNumber(prNumber);
let chain;

// Read the Boyar config from file
const configuration = require(configFilePath);
chain = getBoyarChainConfigurationById(configuration, chainNumber);

// This means we already have a chain in the config, let's just update it's version ref and refresh ports
if (chain !== false) {
    chain.DockerConfig.Tag = targetTag;
    chain.HttpPort = newVacantTCPPort(configuration);
    chain.GossipPort = newVacantTCPPort(configuration);
} else {
    // We need to spawn a new chain for this PR
    let Id = chainNumber;
    let HttpPort = newVacantTCPPort(configuration);
    let GossipPort = newVacantTCPPort(configuration);
    let Tag = targetTag;

    chain = newChainConfiguration({ Id, HttpPort, GossipPort, Tag });
}

const updatedConfiguration = updateChainConfiguration(configuration, chain);

fs.writeFileSync(configFilePath, JSON.stringify(updatedConfiguration, 2, 2));

console.log(chainNumber);
process.exit(0);