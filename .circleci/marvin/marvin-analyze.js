#!/usr/bin/env node

const fs = require('fs');

const pathToJobResults = process.argv[2];

if (!pathToJobResults) {
    console.error('Path to results.json not provided!');
    process.exit(1);
}

const jobResults = require(pathToJobResults);
const {passed} = require('@orbs-network/judge-dredd');

(async function () {
    try {
        const result = await passed({current: jobResults, previous: null, config: null});
        console.log(`Writing job analysis results to disk: ${JSON.stringify(result)}`);
        fs.writeFileSync('workspace/analysis_results.json', JSON.stringify(result, 2, 2));

        if (result.analysis.passed) {
            fs.writeFileSync('workspace/pass_fail.txt', 'PASSED');
            console.log('Marvin analysis determined that the test passed!');
            process.exit(0);
        } else {
            fs.writeFileSync('workspace/pass_fail.txt', 'FAILED');
            console.log('Marvin analysis determined that the test failed.');
            console.log('Reason:', result.analysis.reason);
            // The test failed, but not the process that determined that the test failed..
            // If we fail here, marvin-reporter will not run.
            process.exit(0);
        }
    } catch (err) {
        console.error(err);
        process.exit(2);
    }
})();


