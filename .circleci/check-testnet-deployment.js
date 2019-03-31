#!/usr/bin/env node

const fetch = require('node-fetch');

const TOPOLOGY = 'https://s3.eu-central-1.amazonaws.com/boyar-ci/boyar/config.json';
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

async function getMetricsForNode(ip, vcid) {
    const result = await fetch(`http://${ip}/vchains/${vcid}/metrics`)
        .catch(err => {
            return {
                ok: false,
                err,
            };
        });

    const metrics = await result.json().catch(err => {
        return {
            ok: false,
            err,
        };
    });
    return { ok: true, metrics, ip, vcid };
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
            return getMetricsForNode(ip, chainId);
        }));

        console.log('Got: ', results);

        const nodesSamples = results
            .filter(result => result.ok === true)
            .map(({ ip, metrics, vcid }) => {
                return {
                    ip,
                    vcid,
                    version: metrics['Version.Commit'].Value,
                };
            });

        console.log('Got: ', nodesSamples);
        if (nodesSamples.filter(sample => sample.version === TARGET_HASH).length === nodes.length) {
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
    let closingBlocks = false;

    let previousBlockheights = [], currentBlockheights = [];
    let runsCount = 0;
    const maxRetries = 10; // 5 minutes

    do {
        if (runsCount + 1 > maxRetries) {
            console.log('Max retries limit reached for chainId: ', chainId);
            break;
        }

        runsCount++;
        console.log('Polling all nodes for their blockheight on chainId ', chainId);
        const results = await Promise.all(nodes.map(({ ip }) => {
            return getMetricsForNode(ip, chainId);
        }));

        console.log('Got: ', results);

        const nodesSamples = results
            .filter(result => result.ok === true)
            .map(({ ip, metrics, vcid }) => {
                return {
                    ip,
                    vcid,
                    height: metrics['BlockStorage.BlockHeight'].Value,
                };
            });

        console.log('Got: ', nodesSamples);

        if (previousBlockheights.length > 0) {
            currentBlockheights = nodesSamples;
        } else if (previousBlockheights.length === 0) { // First run
            previousBlockheights = nodesSamples;
        }

        if (blockheightIsAdvancing(previousBlockheights, currentBlockheights) === true) {
            closingBlocks = true;
        }

        console.log('Sleeping for 30 seconds..');
        await new Promise((r) => { setTimeout(r, 30 * 1000); });
    } while (closingBlocks === false);

    return {
        ok: closingBlocks,
        chainId
    };
}

function blockheightIsAdvancing(previous, current) {
    const results = previous.map(sample => {
        const prevHeight = sample.height;
        const current = current.filter(o => o.ip === sample.ip)[0];

        if (current !== undefined) {
            const currentHeight = current.height;
            if (currentHeight > prevHeight) {
                return true;
            }
        }

        return false;
    });

    if (results.filter(r => r === true).length === results.length) {
        return true;
    }
    return false;
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