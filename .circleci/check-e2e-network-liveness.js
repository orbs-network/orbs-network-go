#!/usr/bin/env node

const {eventuallyClosingBlocks} = require('./check-metrics');

const targetChainId = process.argv[2] || 42;
const optionalAlternativePollInterval = process.argv[3] || 60;

if (!targetChainId) {
    console.log('No chainId given!');
    process.exit(1);
}

(async () => {
    const result = await eventuallyClosingBlocks('localhost:8080', optionalAlternativePollInterval );
    if (result === true) {
        console.log(`Blocks are being closed on localhost network (chain ${targetChainId}) !`);
        process.exit(0);
    } else {
        console.error('Chain not closing blocks within the defined window, quiting..');
        process.exit(3);
    }
})();
