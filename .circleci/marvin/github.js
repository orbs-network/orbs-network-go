const fetch = require('node-fetch');
const { sprintf } = require('sprintf-js');
const { createAppAuth } = require("@octokit/auth-app");
const fs = require('fs');
const path = require('path');
const {calculatePct, secondsToHumanReadable} = require('./calc');
const pathToMarvinPrivateKey = path.join(__dirname, 'marvin.pem');
const privateKey = fs.readFileSync(pathToMarvinPrivateKey, 'utf-8');

const auth = createAppAuth({
    id: process.env.MARVIN_APP_ID,
    privateKey,
    installationId: process.env.MARVIN_ORBS_INSTALLATION_ID,
    clientId: process.env.MARVIN_CLIENT_ID,
    clientSecret: process.env.MARVIN_CLIENT_SECRET
});


async function getPullRequest(id) {
    const response = await fetch(`https://api.github.com/repos/orbs-network/orbs-network-go/pulls/${id}`);
    return response.json();
}

async function commentWithMarvinOnGitHub({ id, data, master }) {
    const pullRequest = await getPullRequest(id);

    const commentsUrl = pullRequest.comments_url;

    const commentAsString = createCommentMessage({ data, master });

    const body = {
        body: commentAsString
    };

    const installationAuthentication = await auth({ type: "installation" });
    const { token } = installationAuthentication;

    const commentResult = await fetch(commentsUrl, {
        method: 'post',
        body: JSON.stringify(body),
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `token ${token}`,
        },
    });

    return commentResult.json();
}


function createCommentMessage({ data, master }) {
    const branchStats = getStatsFromRun(data);
    const masterStats = getStatsFromRun(master);
    let message = sprintf("Hey, I've completed a %s endurance test on this code.<br />" +
        "Key metrics for this run (compared to master):<br />", secondsToHumanReadable(branchStats.durationInSeconds));

    message += sprintf(
        "*%f* transactions processed successfully (%+.1f%% from master's %d)<br />",
        branchStats.totalTxCount,
        calculatePct(branchStats.totalTxCount, masterStats.totalTxCount),
        masterStats.totalTxCount
    );

    message += sprintf(
        "*%d* transactions failed (%+.1f%% from master's %d)<br />",
        branchStats.totalTxErrorCount,
        calculatePct(branchStats.totalTxErrorCount, masterStats.totalTxErrorCount),
        masterStats.totalTxErrorCount,
    );

    message += sprintf(
        "Average service time: *%dms* %s (%+.1f%% from master's %dms)<br />",
        branchStats.avgServiceTimeInMillis,
        calculatePct(branchStats.avgServiceTimeInMillis, masterStats.avgServiceTimeInMillis),
        masterStats.avgServiceTimeInMillis,
    );

    message += sprintf(
        "P99 service time: *%dms* %s (%+.1f%% from master's %dms)<br />",
        branchStats.p99ServiceTimeInMillis,
        calculatePct(branchStats.p99ServiceTimeInMillis, masterStats.p99ServiceTimeInMillis),
        masterStats.p99ServiceTimeInMillis,
    );

    message += sprintf(
        "Max service time: *%dms* %s (%+.1f%% from master's %dms)<br />",
        branchStats.maxServiceTimeInMillis,
        calculatePct(branchStats.maxServiceTimeInMillis, masterStats.maxServiceTimeInMillis),
        masterStats.maxServiceTimeInMillis,
    );

    message += sprintf(
        "Memory consumption: *%.2fMB* %s (%+.1f%% from master's %.2fMB)<br />",
        branchStats.totalMemoryConsumptionInMegabytes,
        calculatePct(branchStats.totalMemoryConsumptionInMegabytes, masterStats.totalMemoryConsumptionInMegabytes),
        masterStats.totalMemoryConsumptionInMegabytes,
    );

    return message;
}

function getStatsFromRun(o) {
    const firstUpdate = o.updates[0];
    const lastUpdate = o.updates[o.updates.length - 1];

    const durationInSeconds = o.meta.duration_sec;
    const totalTxCount = lastUpdate.summary.total_tx_count;
    const totalTxErrorCount = lastUpdate.summary.err_tx_count;
    const avgServiceTimeInMillis = lastUpdate.summary.avg_service_time_ms;
    const p99ServiceTimeInMillis = lastUpdate.summary.p99_service_time_ms;
    const maxServiceTimeInMillis = lastUpdate.summary.max_service_time_ms;
    const totalMemoryConsumptionInBytes = parseInt(lastUpdate.summary.max_alloc_mem) - parseInt(firstUpdate.summary.max_alloc_mem);
    const totalMemoryConsumptionInMegabytes = totalMemoryConsumptionInBytes / 1000000;

    return {
        durationInSeconds,
        totalTxCount,
        totalTxErrorCount,
        avgServiceTimeInMillis,
        p99ServiceTimeInMillis,
        maxServiceTimeInMillis,
        totalMemoryConsumptionInBytes,
        totalMemoryConsumptionInMegabytes,
    };
}

module.exports = {
    commentWithMarvinOnGitHub,
};
