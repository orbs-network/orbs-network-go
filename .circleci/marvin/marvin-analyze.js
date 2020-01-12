#!/usr/bin/env node

const fs = require('fs');
const { commentWithMarvinOnGitHub } = require('./github');

const pullRequestUrl = process.env.CI_PULL_REQUESTS || '';

const pathToJobResults = process.argv[2];
const pathToLastMasterJobsResults = process.argv[3];

if (!pathToJobResults) {
    console.error('Path to results.json not provided!');
    process.exit(1);
}

if (!pathToLastMasterJobsResults) {
    console.error('Path to last masters JSON not provided!');
    process.exit(1);
}

const jobResults = require(pathToJobResults);
const lastMasterJobsResults = require(pathToLastMasterJobsResults).data;

// Need to take the latest master job which is in DONE state.
let lastMasterJob = null;

for (let n in lastMasterJobsResults) {
    if (lastMasterJobsResults[n].status == "DONE") {
        lastMasterJob = lastMasterJobsResults[n];
        break;
    }
}

if (!lastMasterJob) {
    console.warn('No latest job from master at this stage');
}

const { passed } = require('@orbs-network/judge-dredd');

(async function () {
    try {
        // Let's see if we can report something to GitHub
        if (pullRequestUrl.length > 0) {
            const prLinkParts = pullRequestUrl.split('/');
            const prNumber = parseInt(prLinkParts[prLinkParts.length - 1]);

            await commentWithMarvinOnGitHub({
                id: prNumber,
                data: jobResults,
                master: lastMasterJob,
            });
        }

        const result = await passed({ current: jobResults, previous: lastMasterJob, config: null });
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


