#!/usr/bin/env node

/*
This script polls ALL chains which are marked for cleanup to verify they've been completely disabled on the testnet

Usage: 

$ testnet-poll-disabled-chains.js - (for cleanup purposes)

*/

const path = require('path');
const fetch = require('node-fetch');

const configFilePath = path.join(process.cwd(), 'config.json');

const configuration = require(configFilePath);
const nodes = configuration.network.filter(({ ip }) => ip !== '54.149.67.22');
const chainIds = configuration.chains
    .filter(chain => chain.Disabled === true)
    .map(({ Id }) => Id);

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
            } else {
                const body = await result.json();
                if ('Description' in body && body.Description === "ORBS blockchain node") {
                    console.log('Node with IP: ', ip, ' has successfully terminated chainId: ', chainId);
                    clearInterval(pid);
                    resolve();
                }
            }

        }, 5000);
    });
}

(async () => {
    await Promise.all(nodes.map(({ ip }) => {
        return Promise.all(chainIds.map(id => pollNetworkRemovalByIP({ ip, chainId: id })));
    }))
        .catch((err) => {
            console.log(err);
            process.exit(1);
        });

    process.exit(0);
})();
