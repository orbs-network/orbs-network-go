#!/bin/bash -x

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion

nvm use v10.14.1

. ./test.common.sh

LAST_COMMIT_MESSAGE=`git --no-pager log --decorate=short --pretty=oneline -n1 $CIRCLE_SHA1`
FAILFAST="-failfast"
TIMEOUT_ACCEPTANCE="20m"
TIMEOUT_REST="10m"
COUNT_ACCEPTANCE=10
COUNT_REST=20

if [[ "${LAST_COMMIT_MESSAGE}" == *"#extraflaky"* ]]; then
    FAILFAST=""
    TIMEOUT_ACCEPTANCE="500m"
    TIMEOUT_REST="500m"
    COUNT_ACCEPTANCE=50
    COUNT_REST=50
fi

if [[ $1 == "NIGHTLY" ]]; then
    curl -o- https://raw.githubusercontent.com/creationix/nvm/v0.33.11/install.sh | bash
    export NVM_DIR="/opt/circleci/.nvm" && . $NVM_DIR/nvm.sh && nvm install v11.2 && nvm use v11.2
    npm install junit-xml-stats -g

    NIGHTLY=1
    echo "performing nightly build (count 1000/2000 , no failfast)"
    FAILFAST=""
    TIMEOUT_ACCEPTANCE="500m"
    TIMEOUT_REST="500m"
    # The number here have been reduced since we use paralleism 6 to run 500 tests in 6 different processes
    COUNT_ACCEPTANCE=50
    COUNT_REST=50
fi

if [ "$CIRCLE_NODE_INDEX" == 0 ] || [ "$CIRCLE_NODE_INDEX" == 1 ] || [ "$CIRCLE_NODE_INDEX" == 2 ] || [ "$CIRCLE_NODE_INDEX" == 3 ] || [ -z "$CIRCLE_NODE_INDEX" ]; then
    go_test_junit_report acceptance ./test/acceptance -count $COUNT_ACCEPTANCE -timeout $TIMEOUT_ACCEPTANCE $FAILFAST -tags "unsafetests"
fi

if [ "$CIRCLE_NODE_INDEX" == 4 ] || [ "$CIRCLE_NODE_INDEX" == 5 ] || [ -z "$CIRCLE_NODE_INDEX" ]; then
    go_test_junit_report blockstorage ./services/blockstorage/test -count $COUNT_ACCEPTANCE -timeout $TIMEOUT_REST $FAILFAST -tags "unsafetests"

    go_test_junit_report internodesync ./services/blockstorage/internodesync -count $COUNT_ACCEPTANCE -timeout $TIMEOUT_REST $FAILFAST -tags "unsafetests"

    go_test_junit_report servicesync ./services/blockstorage/servicesync -count $COUNT_ACCEPTANCE -timeout $TIMEOUT_REST $FAILFAST -tags -tags "unsafetests"

    go_test_junit_report transactionpool ./services/transactionpool/test -count $COUNT_ACCEPTANCE -timeout $TIMEOUT_REST $FAILFAST -tags -tags "unsafetests"
fi
