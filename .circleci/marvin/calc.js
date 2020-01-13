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

module.exports = {
    calculatePct,
    secondsToHumanReadable
};