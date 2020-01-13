const {sprintf} = require('sprintf-js');

function secondsToHumanReadable(seconds) {
    const numHours = Math.floor(((seconds % 31536000) % 86400) / 3600);
    const numMinutes = Math.floor((((seconds % 31536000) % 86400) % 3600) / 60);
    const numSeconds = (((seconds % 31536000) % 86400) % 3600) % 60;
    let humanReadable = '';

    humanReadable += (numHours > 0) ? numHours + " hours " : '';
    humanReadable += (numMinutes > 0) ? numMinutes + " minutes " : '';
    humanReadable += (numSeconds > 0) ? numSeconds + " seconds " : '';

    return humanReadable;
}

function calculatePct(cur, prev) {
    if (cur === prev) {
        return 0;
    }
    if (!prev) { // also covers prev===0
        return 0;
    }
    return 100*((cur-prev) / prev);
}

function createGithubCommentWithMessage({data, master}) {
    const branchStats = getStatsFromRun(data);
    const masterStats = getStatsFromRun(master);

    console.log('Current branch stats: ', JSON.stringify(data));
    console.log('Master branch stats: ', JSON.stringify(master));

    if(!masterStats) {
        console.log('Not sending comment to Github - no master data');
        return "No master data";
    }

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

    if (!o || !o.updates || o.updates.length===0) {
        return null;
    }

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
    calculatePct,
    createGithubCommentWithMessage
};