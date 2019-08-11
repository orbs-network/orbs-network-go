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
    const closedPRs = await getClosedPullRequests();
    let removeCounter = 0;

    for (let key in closedPRs) {
        let aClosedPR = closedPRs[key];
        let aClosedPRChainId = 100000 + aClosedPR.number;

        if (getBoyarChainConfigurationById(configuration, aClosedPRChainId) !== false) {
            console.log(`The chain for PR ${aClosedPR.number} (${aClosedPR.title} / ${aClosedPR.login}) exists and this PR is already closed - marking it for sweeping..`)
            markChainForRemoval(configuration, aClosedPRChainId);
            removeCounter++;
        }
    }

    console.log('Total networks to remove: ', removeCounter);

    if (removeCounter > 0) {
        fs.writeFileSync(configFilePath, JSON.stringify(configuration, 2, 2));
    }

    process.exit(0);
})();
