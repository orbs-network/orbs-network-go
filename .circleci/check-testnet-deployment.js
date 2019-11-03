#!/usr/bin/env node
const path = require('path');

const { waitUntilSync, waitUntilCommit, getBlockHeight } = require('@orbs-network/orbs-nebula/lib/metrics');

const targetChainId = process.argv[2];
const optionalAlternativePublicAPI = process.argv[3] || '';
const optionalAlternativePollInterval = process.argv[4] || 60;

if (!targetChainId) {
    console.log('No chainId given!');
    process.exit(1);
}

let configFilePath, topology, customPublicAPI = false;

if (optionalAlternativePublicAPI.length > 0) {
    console.log('Overriding config.json, will query the public API at the address:', optionalAlternativePublicAPI);
    customPublicAPI = true;
} else {
    configFilePath = path.join(process.cwd(), 'config.json');
    topology = require(configFilePath);
}

async function eventuallyDeployed({ chainId, nodes }) {
    // The correct hash for this chainId is..
    const chain = topology.chains.find(chain => chain.Id === chainId);
    const chainSpecificTargetHash = chain.DockerConfig.Tag;

    // First let's poll the nodes for the correct version
    let versionDeployed = false;

    const promises = nodes.map(({ ip }) => {
        console.log('waiting until commit for chain id: ', chainId, ' and IP: ', ip, ' and commit: ', chainSpecificTargetHash);
        return waitUntilCommit(`${ip}/vchains/${chainId}`, chainSpecificTargetHash);
    });

    try {
        await Promise.all(promises);
        versionDeployed = true;
        console.log('version ', chainSpecificTargetHash, ' deployed on chain id ', chainId);
    } catch (err) {
        console.log(`Version ${chainSpecificTargetHash} might not be deployed on all CI testnet nodes!`);
        console.log('error provided:', err);
    }

    return {
        ok: versionDeployed
    };
}

async function eventuallyClosingBlocks({ chainId, nodes }) {
    let firstEndpoint;

    if (customPublicAPI) {
        firstEndpoint = optionalAlternativePublicAPI;
    } else {
        firstEndpoint = `${nodes[0].ip}/vchains/${chainId}`;
    }

    // First let's get the current blockheight and wait for it to close 5 more blocks
    const currentBlockheight = await getBlockHeight(firstEndpoint);
    console.log('Fetching current blockheight of the network: ', currentBlockheight);

    try {
        let minuteCounter = 0;

        // This is here to avoid 10 minutes without output to the terminal on CircleCI.
        setInterval(async () => {
            minuteCounter++;
            const sampleBlockheight = await getBlockHeight(firstEndpoint);
            console.log(`${minuteCounter}m Network blockheight:  ${sampleBlockheight}`);
        }, 60 * 1000);

        await waitUntilSync(firstEndpoint, currentBlockheight + 5, optionalAlternativePollInterval * 1000, 60 * 1000 * 60);

        return {
            ok: true,
            chainId
        };
    } catch (err) {
        console.log('Network is not advancing for vchain: ', chainId, ' with error: ', err);
        return err;
    }
}

(async () => {
    const ignoreIpsFilterFunction = ({ ip }) => {
        let ips = [], ignoreIps = process.env.TESTNET_IGNORE_IPS;
        if (ignoreIps && ignoreIps.length > 0) {
            ips = ignoreIps.split(',');
        }
        let returnValue = true;

        for (let n in ips) {
            if (ips[n] === ip) {
                returnValue = false;
            }
        }

        return returnValue;
    };

    let nodes, chains;

    if (!customPublicAPI) {
        nodes = topology.network.filter(ignoreIpsFilterFunction);
        chains = topology.chains.map(chain => chain.Id).filter(chainId => chainId === parseInt(targetChainId));

        if (chains.length === 0) {
            console.log('No chains to check!');
            process.exit(2);
        }

        const results = await Promise.all(chains.map((chainId) => eventuallyDeployed({ chainId, nodes })));
        if (results.filter(r => r.ok === true).length === chains.length) {
            console.log('New version deployed successfully on all chains in the testnet');
        } else {
            console.error('New version was not deployed on all nodes within the defined 15 minutes window, quiting..');
            process.exit(2);
        }
    } else {
        chains = [targetChainId];
        nodes = [optionalAlternativePublicAPI];
    }

    const cbResults = await Promise.all(chains.map((chainId) => eventuallyClosingBlocks({ chainId, nodes })));
    if (cbResults.filter(r => r.ok === true).length === chains.length) {
        console.log(`Blocks are being closed on chain ${targetChainId} in the testnet!`);
        process.exit(0);
    } else {
        console.error('Chain not closing blocks within the defined 15 minutes window, quiting..');
        process.exit(3);
    }
})();