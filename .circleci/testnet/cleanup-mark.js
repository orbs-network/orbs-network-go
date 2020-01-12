#!/usr/bin/env node

const path = require('path');
const fs = require('fs');

const configFilePath = path.join(process.cwd(), 'config.json');
const { getClosedPullRequests, getBoyarChainConfigurationById, markChainForRemoval } = require('./boyar-lib');

// Read the Boyar config from file
// We use this const variable as a solid base to start modifying
const configuration = require(configFilePath);

(async function () {
    console.log('Querying closed github PRs...');
    const closedPullRequests = await getClosedPullRequests();

    let removeCounter = 0;
    let updatedConfiguration = configuration;

    for (let key in closedPullRequests) {
        let pr = closedPullRequests[key];
        let chainId = configuration.chains
            .find(c => parseInt(c.Config.prNumber) == pr.number);

        if (!chainId) {
            console.warn(`WARNING: Couldn't find a chain for PR number ${pr.number} (${pr.url})`)
            continue;
        }

        if (getBoyarChainConfigurationById(configuration, chainId) !== false) {
            console.log(`Chain for PR ${pr.number} (PR ${pr.title}, VCHAIN ${chainId}) already closed`);
            console.log(`PR User: ${pr.login}`);
            console.log('--- MARKED for sweeping..');
            console.log('');

            updatedConfiguration = markChainForRemoval(configuration, chainId);
            removeCounter++;
        }
    }

    console.log('Total vChains to remove: ', removeCounter);

    if (removeCounter > 0) {
        fs.writeFileSync(configFilePath, JSON.stringify(updatedConfiguration, 2, 2));
    }

    process.exit(0);
})();
