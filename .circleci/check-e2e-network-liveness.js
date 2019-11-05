#!/usr/bin/env node

const { waitUntilSync, getBlockHeight } = require('@orbs-network/orbs-nebula/lib/metrics');

const targetChainId = process.argv[2] || 42;
const optionalAlternativePollInterval = process.argv[3] || 60;

if (!targetChainId) {
    console.log('No chainId given!');
    process.exit(1);
}

async function eventuallyClosingBlocks({ chainId }) {
    let publicApiEndpoint;

    publicApiEndpoint = 'localhost:8080';

    // First let's get the current blockheight and wait for it to close 5 more blocks
    const currentBlockheight = await getBlockHeight(publicApiEndpoint);
    console.log('Fetching current blockheight of the network: ', currentBlockheight);

    try {
        let minuteCounter = 0;

        // This is here to avoid 10 minutes without output to the terminal on CircleCI.
        setInterval(async () => {
            minuteCounter++;
            const sampleBlockheight = await getBlockHeight(publicApiEndpoint);
            console.log(`${minuteCounter}m Network blockheight:  ${sampleBlockheight}`);
        }, 60 * 1000);

        await waitUntilSync(publicApiEndpoint, currentBlockheight + 5, optionalAlternativePollInterval * 1000, 60 * 1000 * 60);

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
    const result = await eventuallyClosingBlocks({ targetChainId });
    if (result.ok === true) {
        console.log(`Blocks are being closed on localhost network (chain ${targetChainId}) !`);
        process.exit(0);
    } else {
        console.error('Chain not closing blocks within the defined 15 minutes window, quiting..');
        process.exit(3);
    }
})();