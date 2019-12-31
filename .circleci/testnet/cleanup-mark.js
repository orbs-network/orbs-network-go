#!/usr/bin/env node

const path = require('path');
const fs = require('fs');

const configFilePath = path.join(process.cwd(), 'config.json');
const { getClosedPullRequests, getBoyarChainConfigurationById, markChainForRemoval, getAllPrChainIds } = require('./boyar-lib');

// Read the Boyar config from file
// We use this const variable as a solid base to start modifying
const configuration = require(configFilePath);

(async function () {
    console.log('Querying closed github PRs...');
    const closedPRs = await getClosedPullRequests();

    let removeCounter = 0;
    let updatedConfiguration = configuration;

    for (let key in closedPRs) {
        let aClosedPR = closedPRs[key];
        const vChainIds = getAllPrChainIds(aClosedPR.number);

        for (let chainId of vChainIds) {
            if (getBoyarChainConfigurationById(configuration, chainId) !== false) {
                console.log(`Chain for PR ${aClosedPR.number} (PR ${aClosedPR.title}, VCHAIN ${chainId}) already closed`);
                console.log(`PR User: ${aClosedPR.login}`);
                console.log('--- MARKED for sweeping..');
                console.log('');

                updatedConfiguration = markChainForRemoval(configuration, chainId);
                removeCounter++;
            }
        }
    }

    console.log('Total vChains to remove: ', removeCounter);

    if (removeCounter > 0) {
        fs.writeFileSync(configFilePath, JSON.stringify(updatedConfiguration, 2, 2));
    }

    process.exit(0);
})();
