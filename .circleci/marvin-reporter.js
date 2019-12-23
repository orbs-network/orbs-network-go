#!/usr/bin/env node

const execSync = require('child_process').execSync;
const fetch = require('node-fetch');
const fs = require('fs');
const slackKey = process.env.SLACK_MARVIN_NOTIFICATIONS_KEY;
if (!slackKey || slackKey.length === 0) {
    console.log('Environment variable SLACK_MARVIN_NOTIFICATIONS_KEY must be defined!');
    process.exit(1);
}

const slackUrl = `https://hooks.slack.com/services/${slackKey}`;
const jobResultsFile = process.argv[2];

if (!jobResultsFile) {
    console.log('No job results file given!');
    process.exit(1);
}

(async () => {

    const jobResults = await readJobResults(jobResultsFile);
    let msg;
    switch (jobResults.status) {
        case 'DONE':
            msg = await createSlackMessageJobDone(jobResults);
            break;
        case 'ERROR':
            msg = await createSlackMessageJobError(jobResults);
            break;
        default:
            console.log(`Not sending to Slack when jobResult.status is ${jobResults.status}`);
            process.exit(2);
    }

    notifySlack(slackUrl, msg);

})();

async function readJobResults(jobResultsFilePath) {
    return new Promise((resolve, reject) => {
        fs.readFile(jobResultsFilePath, (err, contents) => {
            if (err) {
                reject(err);
                return;
            }
            resolve(JSON.parse(contents));
        });
    });
}

// Read a url from the environment variables
function notifySlack(slackUrl, message) {
    if (slackUrl.length === 0) {
        throw `Failed to notify Slack, missing Slack URL`;
    }

    const baseCommand = `curl -s -X POST --data-urlencode "payload={\\"text\\": \\"${message}\\"}" ${slackUrl}`;
    try {
        execSync(baseCommand);
    } catch (ex) {
        throw `Failed to notify Slack: ${ex}`;
    }
}

// function createSlackMessageJobRunning(jobUpdate) {
//     const startTime = jobUpdate.start_time || '1h';
//     const endTime = jobUpdate.end_time || 'now';
//     return `*--------------------------------------------------------------------------*
// *RUNNING* for *${Math.floor((jobUpdate.runtime || 0) / 1000)}* of ${jobUpdate.duration_sec} seconds, on vchain ${jobUpdate.vchain} with ${jobUpdate.tpm} tx/min.
// *--------------------------------------------------------------------------*
// Sent *${jobUpdate.summary.total_tx_count}* transactions with *${jobUpdate.summary.err_tx_count}* errors.
// Service times (ms): AVG=*${jobUpdate.summary.avg_service_time_ms}* MEDIAN=*${jobUpdate.summary.median_service_time_ms}* P90=*${jobUpdate.summary.p90_service_time_ms}* P99=*${jobUpdate.summary.p99_service_time_ms}* MAX=*${jobUpdate.summary.max_service_time_ms}* STDDEV=*${jobUpdate.summary.stddev_service_time_ms}*
// MinAllocMem: ${jobUpdate.summary.min_alloc_mem} MaxAllocMem: ${jobUpdate.summary.max_alloc_mem} bytes, MaxGoroutines: ${jobUpdate.summary.max_goroutines}
// Errors: ${jobUpdate.error || 'none'}
// <http://ec2-34-222-245-15.us-west-2.compute.amazonaws.com:3000/d/a-3pW-3mk/testnet-results?orgId=1&from=${startTime}&to=${endTime}&var-vchain=${jobUpdate.vchain}&var-validator=All|Grafana> | _Job ID: [${jobUpdate.job_id || 'NA'}] Version: ${jobUpdate.summary.semantic_version || 'NA'} Hash: ${jobUpdate.summary.commit_hash || 'NA'}_`;
//
//     // All: ${JSON.stringify(jobUpdate)}`;
// }

async function createSlackMessageJobDone(jobUpdate) {
    const startTime = jobUpdate.start_time || '1h';
    const endTime = jobUpdate.end_time || 'now';

    const committer = await getCommitterUsernameByCommitHash(jobUpdate.summary.commit_hash);
    const slackUsername = getSlackUsernameForGithubUser(committer);
    const noPassReason = jobUpdate.analysis.passed ? '' : `Reason: ${jobUpdate.analysis.reason}`;

    return `*--------------------------------------------------------------------------*
*DONE* running after *${Math.floor((jobUpdate.runtime || 0) / 1000)}* seconds on vchain ${jobUpdate.vchain} with ${jobUpdate.tpm} tx/min.
*--------------------------------------------------------------------------*
*PASSED? ${jobUpdate.analysis.passed}* ${noPassReason}
Sent *${jobUpdate.summary.total_tx_count}* transactions with *${jobUpdate.summary.err_tx_count}* errors.
Service times (ms): AVG=*${jobUpdate.summary.avg_service_time_ms}* MEDIAN=*${jobUpdate.summary.median_service_time_ms}* P90=*${jobUpdate.summary.p90_service_time_ms}* P99=*${jobUpdate.summary.p99_service_time_ms}* MAX=*${jobUpdate.summary.max_service_time_ms}* STDDEV=*${jobUpdate.summary.stddev_service_time_ms}*
MinAllocMem: ${jobUpdate.summary.min_alloc_mem} MaxAllocMem: ${jobUpdate.summary.max_alloc_mem} bytes, MaxGoroutines: ${jobUpdate.summary.max_goroutines}
Errors: ${jobUpdate.error || 'none'}
<http://ec2-34-222-245-15.us-west-2.compute.amazonaws.com:3000/d/a-3pW-3mk/testnet-results?orgId=1&from=${startTime}&to=${endTime}&var-vchain=${jobUpdate.vchain}&var-validator=All|Grafana> | _Job ID: [${jobUpdate.job_id || 'NA'}] Version: ${jobUpdate.summary.semantic_version || 'NA'} Hash: ${jobUpdate.summary.commit_hash || 'NA'}_
Marvin build triggered by <${slackUsername}>
`;
    // All: ${JSON.stringify(jobUpdate)}`;
}

function createSlackMessageJobError(jobUpdate) {
    jobUpdate = jobUpdate || {};
    jobUpdate.summary = jobUpdate.summary || {};

    return `*[${jobUpdate.summary.semantic_version || ''}]* _[${jobUpdate.job_id || ''}]_ *ERROR:* ${jobUpdate.error||'NA'}`;
}


async function getCommitFromMetricsURL(uri) {
    try {
        const metrics = await fetch(uri);
        return metrics['Version.Commit'].Value;
    } catch (err) {
        return err;
    }
}

async function getCommitterUsernameByCommitHash(commitHash) {
    const uri = `https://api.github.com/repos/orbs-network/orbs-network-go/commits/${commitHash}`;

    try {
        const result = await fetch(uri);
        console.log(`commitHash=${commitHash} ${JSON.stringify(result)}`);
        return result.author ? result.author.login : null;
    } catch (err) {
        return err;
    }
}

function getSlackUsernameForGithubUser(githubLoginHandle) {
    const githubToSlack = {
        'noambergIL': 'UBJ7KDUTG',
        'itamararjuan': 'UC41FJ8LX',
        'IdoZilberberg': 'UAFNVB3PS',
        'amir-arad': 'UPAKXMAAF',
        'electricmonk': 'U94KTLRSR',
        'ronno': 'UB0RYKSFP',
        'vistra': 'UNM6TTUUT',
        'talkol': 'UBW4D5L22',
        'owlen': 'UMDKJ8JCQ',
        'OrLavy': 'UNFC532B1',
        'OdedWx': 'U9KP5DQV9',
        'netoneko': 'U9594T135',
        'jlevison': 'U9VJ8BA2F',
        'gilamran': 'UAGNTRH4K',
        'bolshchikov': 'UFJ8S9G0K',
        'andr444': 'UCX7XHX1A'
    };

    console.log(githubLoginHandle);
    if (!githubLoginHandle || githubLoginHandle.length===0) {
        return 'NA';
    }

    return `@${githubToSlack[githubLoginHandle]}`;
}
