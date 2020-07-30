const request = require("request-promise");

class Metrics {
    constructor(endpoint) {
        this._endpoint = endpoint;
        this._metrics = {};
    }

    async refresh() {
        try {
            const body = await request(`http://${this._endpoint}/metrics`, {
                timeout: 1000,
            });
            this._metrics = JSON.parse(body);
        } catch (e) {
            // Suppressed errors
            console.error(`${e.message}: ${this._endpoint}`);
        }
    }

    _get(names){
        let curr = this._metrics.Payload;
        for (let i = 0;i < names.length; i++) {
            if (curr[names[i]]) {
                curr = curr[names[i]];
            } else {
                return 0;
            }
        }
        return curr;
    }

    getBlockHeight() { return this._get(["BlockStorage", "LastCommitted", "BlockHeight"]) || 0}
}

async function eventuallyClosingBlocks(endpoint, pollingIntervalSeconds) {
    let metrics = new Metrics(endpoint);
    await metrics.refresh();
    pollingIntervalSeconds = pollingIntervalSeconds || 60;
    if (pollingIntervalSeconds <= 0) {
        pollingIntervalSeconds = 60;
    }

    // First let's get the current blockheight and wait for it to close 5 more blocks
    const startBlockHeight = metrics.getBlockHeight();
    console.log('Fetching current blockheight of the network: ', startBlockHeight);

    for (let counter = 0; counter < 60; counter++) {
        await sleep(pollingIntervalSeconds * 1000);
        await metrics.refresh();
        let currentBlockHeight = metrics.getBlockHeight();
        if (currentBlockHeight >= startBlockHeight + 5) {
            console.log(`got to ${currentBlockHeight} - yay!`);
            return true;
        }
        if (counter % 5 === 0) {
            console.log(`try #${counter + 1}: network blockheight:  ${currentBlockHeight}`);
        }
    }
    return false;
}

// general
function sleep(ms) {
    return new Promise(resolve => {
        setTimeout(resolve, ms)
    })
}

module.exports = {
    Metrics, eventuallyClosingBlocks
};
