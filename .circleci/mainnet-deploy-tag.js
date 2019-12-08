#!/usr/bin/env node

const path = require('path');
const fs = require('fs');
const targetTag = process.argv[2];
const configFilePath = path.join(process.cwd(), 'config.json');

if (!targetTag) {
    console.log('No version hash!');
    process.exit(1);
}

const configuration = require(configFilePath);
configuration.chains.forEach((_, index) => {
    configuration.chains[index].DockerConfig.Tag = targetTag;
});

fs.writeFileSync(configFilePath, JSON.stringify(configuration, 2, 2));