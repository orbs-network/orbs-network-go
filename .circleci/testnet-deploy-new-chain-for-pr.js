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
const prNumber = parseInt(prLinkParts[prLinkParts.length - 1]);
const chainNumber = prNumber + 100000;

const configuration = require(configFilePath);

const chainIndex = configuration.chains.findIndex(chain => chain.Id === chainNumber);

if (chainIndex !== -1) {
    // This means we already have a chain in the config, let's just update it's version ref
    configuration.chains[chainIndex].DockerConfig.Tag = targetTag;
} else {
    const lastChain = configuration.chains[configuration.chains.length - 1];

    // Clone the last chain and make modifications on top of it.
    const newChain = Object.assign({}, lastChain);
    const basePort = 9000;
    newChain.DockerConfig.Tag = targetTag;
    newChain.Id = chainNumber;
    newChain.HttpPort = basePort + prNumber;
    newChain.GossipPort = basePort + prNumber + 2;

    configuration.chains.push(newChain);
}

fs.writeFileSync(configFilePath, JSON.stringify(configuration, 2, 2));

console.log(chainNumber);
process.exit(0);