#!/usr/bin/env node

const {
    getSlackUsernameForGithubUser,
    getCommitterUsernameByCommitHash,
    getCommitFromMetricsURL,
    createSlackMessageJobError,
    createSlackMessageJobDone,
    notifySlack,
    readJobAnalysis,
} = require('./reporter-lib');

const slackKey = process.env.SLACK_MARVIN_NOTIFICATIONS_KEY;
if (!slackKey || slackKey.length === 0) {
    console.log('Environment variable SLACK_MARVIN_NOTIFICATIONS_KEY must be defined!');
    process.exit(1);
}

const slackUrl = `https://hooks.slack.com/services/${slackKey}`;
const jobAnalysisFile = process.argv[2];

if (!jobAnalysisFile) {
    console.log('No job analysis file given!');
    process.exit(1);
}

(async () => {
    const job = await readJobAnalysis(jobAnalysisFile);
    console.log(`Will create a Slack message for jobId ${job.jobId}`);
    if (!job.status || !job.summary) {
        job.status = 'ERROR';
        job.error = 'No status or summary property';
    }
    job.summary = job.summary || {};
    let msg;
    switch (job.status) {
        case 'DONE':
            msg = await createSlackMessageJobDone(job);
            break;
        case 'ERROR':
            msg = await createSlackMessageJobError(job);
            break;
        default:
            console.log(`Not sending to Slack because status is ${job.status}`);
            process.exit(2);
    }

    console.log(`[SLACK] Posting message to Slack: ${msg}`);
    notifySlack(slackUrl, msg);
})();

