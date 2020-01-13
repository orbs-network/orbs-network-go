const {calculatePct, createGithubCommentWithMessage} = require('./util');
const {expect} = require('chai');
const {sprintf} = require('sprintf-js');

describe('calculatePct', () => {
    it('should return a positive pct if current more than previous', () => {
        const actual = calculatePct(120, 100);
        expect(actual).to.equal(20);

    });
    it('should return a negative pct if current more than previous', () => {
        const actual = calculatePct(80, 100);
        expect(actual).to.equal(-20);
    });

    it('should return zero pct if current equals previous', () => {
        const actual = calculatePct(120, 120);
        expect(actual).to.equal(0);

    });
    it('should return zero pct if previous is zero or missing', () => {
        let actual = calculatePct(120, 0);
        expect(actual).to.equal(0);
        actual = calculatePct(120);
        expect(actual).to.equal(0);

        console.log(sprintf("%+.1f%%", calculatePct(100, 120)));
        console.log(sprintf("%+.1f", calculatePct(120, 120)));
        console.log(sprintf("%+.1f", calculatePct(120, 100)));

    });
});

let branchData = {
    "meta": {
        "vchain": 826514444,
        "tpm": 18000,
        "duration_sec": 600,
        "client_timeout_sec": 60,
        "gitBranch": "bugfix/marvin/github-comment-fix",
        "target_ips": [
            "35.161.123.97"
        ]
    },
    "updates": [
        {
            "jobId": "20200113_120955_021",
            "executor_port": 4568,
            "executor_pid": 10143,
            "status": "DONE",
            "vchain": 826514444,
            "live_clients": 0,
            "runtime": 605558,
            "duration_sec": 600,
            "tpm": 18000,
            "summary": {
                "total_dur": 197176813,
                "total_tx_count": 161670,
                "err_tx_count": 446,
                "tx_result_types": [
                    {
                        "name": "COMMITTED",
                        "count": 161224
                    },
                    {
                        "name": "failed sending http post: Post http://35.161.123.97/vchains/826514444/api/v1/send-transaction: connection reset by peer",
                        "count": 30
                    },
                    {
                        "name": "http request failed with Content-Type 'text/html': 3c68746d6c3e0d0a3c686561643e3c7469746c653e35303020496e7465726e616c20536572766572204572726f723c2f7469746c653e3c2f686561643e0d0a3c626f64793e0d0a3c63656e7465723e3c68313e35303020496e7465726e616c20536572766572204572726f723c2f68313e3c2f63656e7465723e0d0a3c68723e3c63656e7465723e6e67696e782f312e31372e373c2f63656e7465723e0d0a3c2f626f64793e0d0a3c2f68746d6c3e0d0a",
                        "count": 396
                    },
                    {
                        "name": "failed sending http post: Post http://35.161.123.97/vchains/826514444/api/v1/send-transaction: EOF",
                        "count": 20
                    }
                ],
                "max_service_time_ms": 2170,
                "stddev_service_time_ms": 339,
                "avg_service_time_ms": 1219,
                "median_service_time_ms": 1305,
                "p90_service_time_ms": 1460,
                "p95_service_time_ms": 1518,
                "p99_service_time_ms": 1704,
                "max_alloc_mem": 0,
                "max_goroutines": 0,
                "semantic_version": "v1.3.6-57488c41",
                "commit_hash": "57488c41376813eba9a9e2e3cbf50006bcbafe77",
                "total_count": 161670
            },
            "start_time": "2020-01-13T12:09:56.359Z",
            "current_time": "2020-01-13T12:20:01.917Z",
            "end_time": "2020-01-13T12:20:01.917Z",
            "client_cmd": "docker run -t --rm endurance:client ./client 826514444,35.161.123.97 20200113_120955_021_9,14,18000"
        }
    ]
};

let masterData = {};

describe('createGithubCommentWithMessage', () => {
    it('should return the text of the comment', () => {
        const actual = createGithubCommentWithMessage({data: branchData, master: masterData});
        console.log(actual);
    });
});