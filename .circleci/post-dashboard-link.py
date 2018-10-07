import os
import urllib2
import json
import re

TOKEN = os.environ['GITHUB_COMMENTS_TOKEN']

def get_pull_requests():
    url = 'https://api.github.com/repos/orbs-network/orbs-network-go/pulls'
    data = json.load(urllib2.urlopen(url))
    return map(lambda node: {'number': node['number'], 'branch': node['head']['ref'], 'repo': node['head']['repo']['full_name']}, data)

def orbs_network_only(pr):
    return pr['repo'] == 'orbs-network/orbs-network-go'

def get_pull_request_comments(pr):
    url = 'https://api.github.com/repos/orbs-network/orbs-network-go/issues/' + str(pr['number']) + '/comments'
    data = json.load(urllib2.urlopen(url))
    return map(lambda node: {'id': node['id'], 'body': node['body']}, data)

def has_dashboard_comment(comment):
    return re.match('.*app.redash.io.*', comment['body'])

def post_dashboard_link(pr):
    formatted_branch = pr['branch'].replace('/', '-')
    dashboard_url = 'https://app.redash.io/orbs/dashboard/ci?p_test=acceptance&p_branch=' + formatted_branch
    url = 'https://api.github.com/repos/orbs-network/orbs-network-go/issues/' + str(pr['number']) + '/comments'

    data = json.dumps({'body': 'Metrics dashboard: ' + dashboard_url + '\n\nThis is an automated message.'})
    req = urllib2.Request(url, data, {'Content-Type': 'application/json', 'Authorization': 'token ' + TOKEN})
    f = urllib2.urlopen(req)
    response = f.read()
    f.close()
    return response

def update_pull_request(pr):
    print 'Found pull request', pr['number']
    comments = get_pull_request_comments(pr)

    if any(map(has_dashboard_comment, comments)):
        print 'Comment already exists, skipping'
    else:
        print 'Posting dashboard link'
        post_dashboard_link(pr)

if __name__ == '__main__':
    pull_requests = filter(orbs_network_only, get_pull_requests())
    map(update_pull_request, pull_requests)
