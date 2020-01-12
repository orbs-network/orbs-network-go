'use strict';

const {createSlackMessageJobDone} = require('./reporter-lib');

describe('slack messages', () => {
    it('should create a Slack report from job update JSON', () => {

        const jobUpdate = {
            "jobId": "20200109_195647_055",
            "executor_port": 4568,
            "executor_pid": 20037,
            "status": "DONE",
            "vchain": 323142232,
            "live_clients": 0,
            "runtime": 3603090,
            "duration_sec": 3600,
            "tpm": 18000,
            "analysis":{
                passed: true,
            },
            "summary": {
                "total_dur": 1270381987,
                "total_tx_count": 971340,
                "err_tx_count": 54272,
                "tx_result_types": [
                    {
                        "name": "COMMITTED",
                        "count": 917069
                    },
                    {
                        "name": "failed sending http post: Post http://35.161.123.97/vchains/323142232/api/v1/send-transaction: connection reset by peer",
                        "count": 2112
                    },
                    {
                        "name": "http request failed with Content-Type 'text/html': 3c68746d6c3e0d0a3c686561643e3c7469746c653e35303020496e7465726e616c20536572766572204572726f723c2f7469746c653e3c2f686561643e0d0a3c626f64793e0d0a3c63656e7465723e3c68313e35303020496e7465726e616c20536572766572204572726f723c2f68313e3c2f63656e7465723e0d0a3c68723e3c63656e7465723e6e67696e782f312e31372e363c2f63656e7465723e0d0a3c2f626f64793e0d0a3c2f68746d6c3e0d0a",
                        "count": 49003
                    },
                    {
                        "name": "failed sending http post: Post http://35.161.123.97/vchains/323142232/api/v1/send-transaction: EOF",
                        "count": 3155
                    },
                    {
                        "name": "failed sending http post: Post http://35.161.123.97/vchains/323142232/api/v1/send-transaction: http: server closed idle connection",
                        "count": 1
                    }
                ],
                "max_service_time_ms": 8811,
                "stddev_service_time_ms": 512,
                "avg_service_time_ms": 1307,
                "median_service_time_ms": 1335,
                "p90_service_time_ms": 1686,
                "p95_service_time_ms": 1982,
                "p99_service_time_ms": 2563,
                "max_alloc_mem": 0,
                "max_goroutines": 0,
                "semantic_version": "v1.3.6-6f26aaaf",
                "commit_hash": "6f26aaaf8798e8d0bbdda0795a400ce1ad6d8713",
                "total_count": 971340
            },
            "start_time": "2020-01-09T19:56:48.776Z",
            "current_time": "2020-01-09T20:56:51.866Z",
            "end_time": "2020-01-09T20:56:51.866Z",
            "client_cmd": "docker run -t --rm endurance:client ./client 323142232,35.161.123.97 20200109_195647_055_55,21,18000"
        };

        const slackMessage = createSlackMessageJobDone(jobUpdate);
        console.log(slackMessage);

    });
});