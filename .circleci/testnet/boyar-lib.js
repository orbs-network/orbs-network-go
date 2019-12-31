const fetch = require('node-fetch');
const crypto = require('crypto');

function getChainIdFromBranchName(branch) {
    let v = 0;

    for (let k in branch) {
        v = v ^ branch[k].charCodeAt(0) << k % 64;
    }

    return Math.abs(v);
}

function getBoyarChainConfigurationById(configuration, chainId) {
    const chainIndex = configuration.chains.findIndex(chain => chain.Id === chainId);
    return (chainIndex !== -1) ? configuration.chains[chainIndex] : false;
}

function newChainConfiguration({ Id, HttpPort, GossipPort, Tag }) {
    return {
        Id,
        HttpPort,
        GossipPort,
        Resources: {
            Limits: {
                Memory: 1024,
                CPUs: 1
            },
            Reservations: {
                Memory: 1,
                CPUs: 0.001
            }
        },
        DockerConfig: {
            ContainerNamePrefix: "orbs-network",
            Image: "727534866935.dkr.ecr.us-west-2.amazonaws.com/orbs-network-v1",
            Tag,
            Pull: true
        },
        Config: {
            'genesis-validator-addresses': [
                "a328846cd5b4979d68a8c58a9bdfeee657b34de7",
                "d27e2e7398e2582f63d0800330010b3e58952ff6",
                "6e2cb55e4cbe97bf5b1e731d51cc2c285d83cbf9",
                "c056dfc0d1fbc7479db11e61d1b0b57612bf7f17",
                "e93269e5752cec83b758cbceaeddf4d40e1556c3"
            ],
            'ethereum-endpoint': "http://ropsten.infura.io/v3/f4cc67780a124ad9a6953083ed3c5991",
            'ethereum-finality-blocks-component': 10,
            'active-consensus-algo': 2,
            'logger-file-truncation-interval': "24h",
            profiling: true,
            'processor-sanitize-deployed-contracts': false,
            reload: "1"
        }
    };
}

function randomInt(low, high) {
    return Math.floor(Math.random() * (high - low) + low)
}

function isPortUnique(configuration, port) {
    let returnValue = true;
    configuration.chains.forEach(({ HttpPort, GossipPort }) => {
        if (parseInt(HttpPort) === port || parseInt(GossipPort) === port) {
            returnValue = false;
        }
    });
    return returnValue;
}

function newVacantTCPPort(configuration) {
    let randomPort;

    do {
        randomPort = randomInt(4000, 20000);
    } while (!isPortUnique(configuration, randomPort));

    return randomPort;
}

function removeChainConfigurationById(configuration, chainId) {
    const chainIndex = configuration.chains.findIndex(chain => chain.Id === chainId);
    if (chainIndex !== -1) {
        configuration.chains.splice(chainIndex, 1);
    }
    return configuration;
}

function markChainForRemoval(configuration, chainId) {
    const chainIndex = configuration.chains.findIndex(chain => chain.Id === chainId);
    if (chainIndex !== -1) {
        configuration.chains[chainIndex].Disabled = true;
    }
    return configuration;
}

function updateChainConfiguration(configuration, chain) {
    // Incase the chain configuration already exists, let's just remove it.
    removeChainConfigurationById(configuration, chain.Id)
    configuration.chains.push(chain)
    return configuration
}

function getAllPrChainIds(prNumber) {
    const chainIds = [];
    for (let aChainType in vChainIdOffsetByType) {
        chainIds.push(getPrChainId(prNumber, aChainType))
    }
    return chainIds;
}

// to get the first vChain id allocated to prNumber PR, pass undefined to vChainType
function getPrChainId(prNumber, vChainType) {
    let offset = vChainIdOffsetByType[vChainType];

    if (offset === undefined) {
        throw `Unknown virtual chain type ${vChainType}`
    }

    return 100000 + (prNumber * vChainTypesCount + offset) % 1000000;
}

async function getClosedPullRequests(page = 1) {
    const response = await fetch(`https://api.github.com/repos/orbs-network/orbs-network-go/pulls?state=closed&per_page=100&page=${page}`);
    const closedPRs = await response.json();
    return closedPRs.map(({ number, title, user: { login } }) => ({ number, title, login }));
}

module.exports = {
    getClosedPullRequests,
    newChainConfiguration,
    getChainIdFromBranchName,
    getPrChainId,
    getAllPrChainIds,
    getBoyarChainConfigurationById,
    updateChainConfiguration,
    newVacantTCPPort,
    removeChainConfigurationById,
    isPortUnique,
    markChainForRemoval,
};
