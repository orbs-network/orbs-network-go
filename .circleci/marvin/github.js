const fetch = require('node-fetch');
const {createAppAuth} = require("@octokit/auth-app");
const fs = require('fs');
const path = require('path');
const {createGithubCommentWithMessage} = require('./util');
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

async function commentWithMarvinOnGitHub({id, data, master}) {
    const pullRequest = await getPullRequest(id);

    const commentsUrl = pullRequest.comments_url;

    const commentAsString = createGithubCommentWithMessage({data, master});

    const body = {
        body: commentAsString
    };

    const installationAuthentication = await auth({type: "installation"});
    const {token} = installationAuthentication;

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


module.exports = {
    commentWithMarvinOnGitHub,
};
