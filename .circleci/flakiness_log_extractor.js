const fs = require('fs');
const path = require('path');

const targetFilename = path.join(process.cwd(), process.argv[2]);
const result = fs.readFileSync(targetFilename);

const logOutput = result.toString();

const logOutputAsArray = logOutput.split('\n');

let index = false;

for (let k = logOutputAsArray.length - 1; k > 0; k--) {
    if (logOutputAsArray[k - 1].substr(0, 7) === '=== RUN' &&
        logOutputAsArray[k].substr(0, 9) === '--- FAIL:') {
        index = k - 1;
        break;
    }
}

const onlyFailedLog = logOutputAsArray.slice(index, logOutputAsArray.length);
console.log(onlyFailedLog.join('\n'));