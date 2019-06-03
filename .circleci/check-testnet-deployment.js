#!/usr/bin/env node

const { waitUntilSync, waitUntilCommit, getBlockHeight, getCommit } = require('@orbs-network/orbs-nebula/lib/metrics');

const topology = require('./config.json');
const TARGET_HASH = process.argv[2];

if (!TARGET_HASH) {
    console.log('No target hash to check');
    process.exit(1);
}

async function eventuallyDeployed({ chainId, nodes }) {
    // First let's poll the nodes for the correct version
    let versionDeployed = false;

    const promises = nodes.map(({ ip }) => {
        return waitUntilCommit(`${ip}/vchains/${chainId}`, TARGET_HASH);
    });

    try {
        await Promise.all(promises);
        versionDeployed = true;
    } catch (err) {
        console.log(`Version ${TARGET_HASH} might not be deployed on all CI testnet nodes!`);
        console.log('error provided:', err);
    }

    return {
        ok: versionDeployed
    };
}

async function eventuallyClosingBlocks({ chainId, nodes }) {
    const firstEndpoint = `${nodes[0].ip}/vchains/${chainId}`;

    // First let's get the current blockheight and wait for it to close 5 more blocks
    const currentBlockheight = await getBlockHeight(firstEndpoint);

    try {
        await waitUntilSync(firstEndpoint, currentBlockheight + 5);

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
    const nodes = topology.network;
    const chains = topology.chains.map(chain => chain.Id);

    const results = await Promise.all(chains.map((chainId) => eventuallyDeployed({ chainId, nodes })));
    if (results.filter(r => r.ok === true).length === chains.length) {
        console.log('New version deployed successfully on all chains in the testnet');
    } else {
        console.error('New version was not deployed on all nodes within the defined 15 minutes window, quiting..');
        process.exit(2);
    }

    const results = await Promise.all(chains.map((chainId) => eventuallyClosingBlocks({ chainId, nodes })));
    if (results.filter(r => r.ok === true).length === chains.length) {
        console.log('Blocks are being closed on all chains in the testnet!');
        process.exit(0);
    } else {
        console.error('Not all chains are closing blocks after the new version was deployed within the defined 15 minutes window, quiting..');
        process.exit(3);
    }
})();