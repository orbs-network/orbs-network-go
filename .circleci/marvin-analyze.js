#!/usr/bin/env node

const fs = require('fs');

const pathToJobResults = process.argv[2];

if (!pathToJobResults) {
    console.error('Path to results.json not provided!');
    process.exit(1);
}

const jobResults = require(pathToJobResults);
const { passed } = require('@orbs-network/judge-dredd');

(async function () {
    try {
        const result = await passed(jobResults);
        console.log(`Writing job analysis results to disk: ${JSON.stringify(result)}`);
        fs.writeFileSync('workspace/analysis_results.json', JSON.stringify(result, 2, 2));

        if (result.passed) {
            console.log('Marvin analysis is successful!');
            process.exit(0);
        } else {
            console.log('Marvin analysis found some errors. Reason:', passed.reason);
            // The test failed, but not the process that determined that the test failed..
            process.exit(0);
        }
    } catch (err) {
        console.error(err);
        process.exit(2);
    }
})();


