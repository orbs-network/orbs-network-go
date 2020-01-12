const fetch = require('node-fetch');
const { sprintf } = require('sprintf-js');
const { createAppAuth } = require("@octokit/auth-app");
const fs = require('fs');
const path = require('path');
const pathToMarvinPrivateKey = path.join(__dirname, 'marvin.pem');
const privateKey = fs.readFileSync(pathToMarvinPrivateKey, 'utf-8');

const auth = createAppAuth({
    id: process.env.MARVIN_APP_ID,
    privateKey,
    installationId: process.env.MARVIN_ORBS_INSTALLATION_ID,
    clientId: process.env.MARVIN_CLIENT_ID,
    clientSecret: process.env.MARVIN_CLIENT_SECRET
});

function secondsToHumanReadable(seconds) {
    var numhours = Math.floor(((seconds % 31536000) % 86400) / 3600);
    var numminutes = Math.floor((((seconds % 31536000) % 86400) % 3600) / 60);
    var numseconds = (((seconds % 31536000) % 86400) % 3600) % 60;
    let humanReadable = '';

    humanReadable += (numhours > 0) ? numhours + " hours " : '';
    humanReadable += (numminutes > 0) ? numminutes + " minutes " : '';
    humanReadable += (numseconds > 0) ? numseconds + " seconds " : '';

    return humanReadable;
}

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

function calculateSign(c, m) {
    if (c > m) {
        const gain = c - m;
        const gainPercent = gain / m;
        const textualGain = sprintf("%.1f", gainPercent * 100);

        return `*+%${textualGain}*`;
    } else if (m > c) {
        const loss = m - c;
        const lossPercent = loss / c;
        const textualLoss = sprintf("%.1f", lossPercent * 100);

        return `*-%${textualLoss}*`;
    } else {
        return "";
    }
}

function createCommentMessage({ data, master }) {
    const branchStats = getStatsFromRun(data);
    const masterStats = getStatsFromRun(master);
    let message = sprintf("Hey, I've completed an endurance test on this code <br />" +
        "for a duration of %s <br />" +
        "Key metrics from the run (compared to master):<br />", secondsToHumanReadable(branchStats.durationInSeconds));

    message += sprintf(
        "*%f* transactions processed successfully %s(%f)<br />",
        branchStats.totalTxCount,
        calculateSign(branchStats.totalTxCount, masterStats.totalTxCount),
        masterStats.totalTxCount
    );

    message += sprintf(
        "*%d* transactions failed %s(%d)<br />",
        branchStats.totalTxErrorCount,
        calculateSign(branchStats.totalTxErrorCount, masterStats.totalTxErrorCount),
        masterStats.totalTxErrorCount,
    );

    message += sprintf(
        "average service time: *%dms* %s(%dms)<br />",
        branchStats.avgServiceTimeInMilis,
        calculateSign(branchStats.avgServiceTimeInMilis, masterStats.avgServiceTimeInMilis),
        masterStats.avgServiceTimeInMilis,
    );

    message += sprintf(
        "Memory consumption: *%.2fMB* %s(%.2fMB)<br />",
        branchStats.totalMemoryConsumptionInMegabytes,
        calculateSign(branchStats.totalMemoryConsumptionInMegabytes, masterStats.totalMemoryConsumptionInMegabytes),
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
    const avgServiceTimeInMilis = lastUpdate.summary.avg_service_time_ms;
    const totalMemoryConsumptionInBytes = parseInt(lastUpdate.summary.max_alloc_mem) - parseInt(firstUpdate.summary.max_alloc_mem);
    const totalMemoryConsumptionInMegabytes = totalMemoryConsumptionInBytes / 1000000;

    return {
        durationInSeconds,
        totalTxCount,
        totalTxErrorCount,
        avgServiceTimeInMilis,
        totalMemoryConsumptionInBytes,
        totalMemoryConsumptionInMegabytes,
    };
}

module.exports = {
    commentWithMarvinOnGitHub,
};
