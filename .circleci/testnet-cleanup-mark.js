#!/usr/bin/env node

const path = require('path');
const fs = require('fs');

const configFilePath = path.join(process.cwd(), 'config.json');
const { getClosedPullRequests, getPrChainNumber, getBoyarChainConfigurationById, markChainForRemoval } = require('./boyar-lib');

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
        let aClosedPRChainId = getPrChainNumber(aClosedPR.number);

        if (getBoyarChainConfigurationById(configuration, aClosedPRChainId) !== false) {
            console.log(`Chain for PR ${aClosedPR.number} (${aClosedPR.title}) already closed`);
            console.log(`PR User: ${aClosedPR.login}`)
            console.log('--- MARKED for sweeping..');
            console.log('');
            updatedConfiguration = markChainForRemoval(configuration, aClosedPRChainId);
            removeCounter++;
        }
    }

    console.log('Total networks to remove: ', removeCounter);

    if (removeCounter > 0) {
        fs.writeFileSync(configFilePath, JSON.stringify(updatedConfiguration, 2, 2));
    }

    process.exit(0);
})();
