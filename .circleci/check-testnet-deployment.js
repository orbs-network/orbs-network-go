#!/usr/bin/env node

const { waitUntilSync, getBlockHeight, getCommit } = require('@orbs-network/orbs-nebula/lib/metrics');
const fetch = require('node-fetch');

const TOPOLOGY = 'https://s3.us-west-2.amazonaws.com/boyar-testnet-bootstrap/boyar/config.json';
const TARGET_HASH = process.argv[2];

if (!TARGET_HASH) {
    console.log('No target hash to check');
    process.exit(1);
}

async function getTopology() {
    const result = await fetch(TOPOLOGY);
    const topology = await result.json();
    return topology;
}

async function eventuallyDeployed({ chainId, nodes }) {
    // First let's poll the nodes for the correct version
    // Until we meet the correct version on all nodes or we bail by the timeout defined
    let versionDeployed = false;
    let runsCount = 0;
    const maxRetries = 10; // 5 minutes

    do {
        if (runsCount + 1 > maxRetries) {
            console.log('Max retries limit reached for chainId: ', chainId);
            break;
        }

        runsCount++;
        console.log('Polling all nodes for their version on chainId ', chainId);
        const results = await Promise.all(nodes.map(({ ip }) => {
            return getCommit(`${ip}/vchains/${chainId}`);
        }))
            .catch(err => {
                console.log('failed getting the commit hash of one of the nodes for some reason');
                console.log('error provided:', err);
            });

        console.log('Got: ', results);
        if (results.filter(version => version === TARGET_HASH).length === nodes.length) {
            // Version updated on all nodes
            versionDeployed = true;
            continue;
        }

        console.log('Sleeping for 30 seconds..');
        await new Promise((r) => { setTimeout(r, 30 * 1000); });
    } while (versionDeployed === false);

    return {
        ok: versionDeployed
    };
}

async function eventuallyClosingBlocks({ chainId, nodes }) {
    const firstEndpoint = `${nodes[0].ip}/vchains/${chainId}`;

    // First let's get the current blockheight and wait for it to close another 50 blocks.
    const currentBlockheight = await getBlockHeight(firstEndpoint);

    try {
        await waitUntilSync(firstEndpoint, currentBlockheight + 50);

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
    const topology = await getTopology();
    const nodes = topology.network;
    const chains = topology.chains.map(chain => chain.Id);

    const results = await Promise.all(chains.map((chainId) => eventuallyDeployed({ chainId, nodes })));
    if (results.filter(r => r.ok === true).length === chains.length) {
        console.log('New version deployed successfully on all chains in the testnet');
    } else {
        console.error('New version was not deployed on all nodes within the defined 5 minutes window, quiting..');
        process.exit(2);
    }

    const results = await Promise.all(chains.map((chainId) => eventuallyClosingBlocks({ chainId, nodes })));
    if (results.filter(r => r.ok === true).length === chains.length) {
        console.log('Blocks are being closed on all chains in the testnet!');
        process.exit(0);
    } else {
        console.error('Not all chains are closing blocks after the new version was deployed within the defined 5 minutes window, quiting..')
        process.exit(3);
    }
})();