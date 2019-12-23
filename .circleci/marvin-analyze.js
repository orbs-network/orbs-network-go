#!/usr/bin/env node

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

        if (result.passed) {
            console.log('Marvin analysis is successful!');
            process.exit(0);
        } else {
            console.error('Marvin analysis found some errors:');
            console.log('Reason:', passed.reason);
            process.exit(3);
        }
    } catch (err) {
        console.error(err);
        process.exit(2);
    }
})();


