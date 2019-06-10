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

// The namespace 100000 is PR chains teritory
const prLinkParts = githubPRLink.split('/');
const prNumber = parseInt(prLinkParts[prLinkParts.length - 1]) + 100000;

const configuration = require(configFilePath);
const lastChain = configuration.chains[configuration.chains.length - 1];

// Clone the last chain and make modifications on top of it.
const newChain = Object.assign({}, lastChain);
newChain.DockerConfig.Tag = targetTag;
newChain.Id = prNumber;
newChain.HttpPort = newChain.HttpPort + 2;
newChain.GossipPort = newChain.GossipPort + 2;

configuration.chains.push(newChain);

fs.writeFileSync(configFilePath, JSON.stringify(configuration, 2, 2));

console.log(prNumber);
process.exit(0);