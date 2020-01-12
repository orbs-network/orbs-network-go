const { describe, it } = require('mocha');
const { expect } = require('chai');
const { isPortUnique, newVacantTCPPort, getChainIdFromBranchName } = require('./../boyar-lib');
const branches = require('./../fixtures/branches');

describe('boyar library tests', () => {
    it('should identify a port as unique correctly', () => {
        const configuration = {
            chains: [
                {
                    HttpPort: 1000,
                    GossipPort: 1001,
                },
                {
                    HttpPort: 1003,
                    GossipPort: 1005,
                },
            ]
        }

        expect(isPortUnique(configuration, 1002)).to.equal(true)
        expect(isPortUnique(configuration, 1000)).to.equal(false)
    })

    it('should generate a unique (32bit integer or less) vchainId per a given branch name', async () => {        
        const ids = [];

        for (let n in branches) {
            const branchName = branches[n].name;
            ids.push({
                branchName,
                id: getChainIdFromBranchName(branchName)
            });
        }

        const idHistory = [];
        const dups = [];

        for (let key in ids) {
            if (idHistory.includes(ids[key].id)) {
                dups.push(ids[key]);
            }
            idHistory.push(ids[key].id);
        }

        expect(dups).to.eql([]);
    });

    it('should provide a vacant port', () => {
        const configuration = {
            chains: [
                {
                    HttpPort: 1000,
                    GossipPort: 1001,
                },
                {
                    HttpPort: 1003,
                    GossipPort: 1005,
                },
            ]
        }

        const newPort = newVacantTCPPort(configuration)
        expect([1000, 1001, 1003, 1005]).to.not.include(newPort)
    })
})
