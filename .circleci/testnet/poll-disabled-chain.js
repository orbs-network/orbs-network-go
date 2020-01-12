#!/usr/bin/env node

/*
This script polls a specific chain to verify it's been completely disabled on the provided network

Usage: 

$ testnet-poll-disabled-chain.js <ChainID>

Examples:

$ testnet-poll-disabled-chain.js 1104832

*/

const path = require('path');
const fetch = require('node-fetch');

const targetChainId = process.argv[2];
const configFilePath = path.join(process.cwd(), 'config.json');

if (!targetChainId) {
    console.log('No chainId given!');
    process.exit(1);
}

const configuration = require(configFilePath);
const nodes = configuration.network.filter(({ ip }) => ip !== '54.149.67.22');

function pollNetworkRemovalByIP({ ip, chainId }) {
    let attempts = 0; const maxAttempts = 60; // 5 minutes
    return new Promise((resolve, reject) => {
        let pid = setInterval(async () => {
            attempts++;
            if (attempts >= maxAttempts) {
                clearInterval(pid);
                reject(`Could not identify that chain ${chainId} was removed from IP: ${ip}`);
                return;
            }
            const url = `http://${ip}/vchains/${chainId}/metrics`;
            console.log('Calling ', url, ` attempt #${attempts}`);
            const result = await fetch(url);

            console.log('Got response status: ', result.status);
            if (result.status === 404) {
                console.log('Node with IP: ', ip, ' has successfully terminated chainId: ', chainId);
                clearInterval(pid);
                resolve();
            }
        }, 5000);
    });
}

(async () => {
    await Promise.all(nodes.map(({ ip }) => {
        return pollNetworkRemovalByIP({ ip, chainId: targetChainId });
    }))
        .catch((err) => {
            console.log(err);
            process.exit(1);
        });

    process.exit(0);
})();

