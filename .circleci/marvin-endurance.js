#!/usr/bin/env node

const fetch = require('node-fetch');

const marvinUrl = process.env.MARVIN_URL;
let vchain = process.argv[2];
const targetIp = process.argv[3];
const writeTargetPath = process.argv[4];
const fs = require('fs');

function printUsage() {
    console.log('marvin-endurance usage: ');
    console.log('./marvin-endurance.js <vchain> <targetIp>');
    console.log('Example:');
    console.log(' ./marvin-endurance.js 102030 1.2.3.4');
}

if (!marvinUrl) {
    console.error('Cannot query Marvin API without knowing it\'s address. Please set MARVIN_URL to something');
    process.exit(2);
}

if (!vchain) {
    console.error('vchain not provided, cannot call marvin without a vchain!');
    printUsage();
    process.exit(1);
}

vchain = parseInt(vchain);

if (!targetIp) {
    console.error('targetIp not provided, cannot call marvin without a targetIp!');
    printUsage();
    process.exit(1);
}

if (!writeTargetPath) {
    console.warn('WARN: the writeTargetPath is not provided, output to the CircleCI workspace is disabled');
    console.warn('To enable run this command like so:');
    console.warn(` ./marvin-endurance.js ${vchain} ${targetIp} workspace-dir/job_id`);
}

(async function () {
    const body = {
        vchain,
        tpm: 60,
        duration_sec: 60,
        client_timeout_sec: 120,
        target_ips: [targetIp]
    };

    console.log('Sending marvin a request to start a new endurance test (transferFrenzy)..');

    const result = await fetch(`${marvinUrl}/jobs/start/transferFrenzy`, {
        method: 'post',
        body: JSON.stringify(body),
        headers: { 'Content-Type': 'application/json' },
    });

    const responseAsJson = await result.json();

    console.log(`Marvin response (HTTP ${result.status}): `);
    const { jobId } = responseAsJson;
    console.log('Received jobId: ', jobId);

    const pollingBoolRes = await waitUntilDone({
        jobId,
        timeoutInSeconds: 120,
        acceptableDurationInSeconds: body.duration_sec
    });

    if (pollingBoolRes) {
        console.log('Marvin test completed');

        if (typeof writeTargetPath == 'string' && writeTargetPath.length > 3) {
            fs.writeFileSync(writeTargetPath, jobId);
            console.log(`jobId written to workspace at ${writeTargetPath}`);
        }

        process.exit(0);
    } else {
        console.log('Marvin test did not complete within the alloted time frame!');
        process.exit(100);
    }
})();

function pSleep(s) {
    return new Promise((r) => { setTimeout(r, s * 1000) });
}

function nowInUnix() {
    return Math.floor(Date.now() / 1000);
}

async function waitUntilDone({ jobId, timeoutInSeconds = 30, acceptableDurationInSeconds = 100 }) {
    const startTime = nowInUnix();
    const maxAllowedEndTime = startTime + acceptableDurationInSeconds + timeoutInSeconds;
    let tick = 0;
    let returnValue = false;

    do {
        tick++;

        const res = await fetch(`${marvinUrl}/jobs/${jobId}/status`);
        const response = await res.json();

        const latestSummary = response.updates[response.updates.length - 1].summary;

        console.log('');
        console.log(`------------------------------------------`);
        console.log(`Status #${tick}: ${response.status}`);
        console.log(`Updates so far: ${response.updates.length}`);
        console.log(`Total Successful Transactions: ${latestSummary.total_tx_count}`);
        console.log(`Total Errornous Transactions: ${latestSummary.err_tx_count}`);
        console.log(`Average service time: ${latestSummary.avg_service_time_ms}`);
        console.log(`------------------------------------------`);
        console.log('');

        if (response.status == 'DONE') {
            returnValue = true;
            break;
        }

        await pSleep(10);
    } while (nowInUnix() <= maxAllowedEndTime);

    return returnValue;
}
