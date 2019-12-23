#!/usr/bin/env node

const execSync = require('child_process').execSync;
const slackKey = process.env.SLACK_MARVIN_NOTIFICATIONS_KEY;
if (!slackKey || slackKey.length === 0) {
    console.log('Environment variable SLACK_MARVIN_NOTIFICATIONS_KEY must be defined!');
    process.exit(1);
}
const slackUrl = `https://hooks.slack.com/services/${slackKey}`;
const message = createSlackMessage();

const baseCommand = `curl -s -X POST --data-urlencode "payload={\\"text\\": \\"${message}\\"}" ${slackUrl}`;
try {
    execSync(baseCommand);
} catch (ex) {
    console.log(`Failed to notify Slack: ${ex}`);
}

function createSlackMessage() {
    return "Hello from marvin_reporter!";
}