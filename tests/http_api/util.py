import sys
import requests as rq
import time
import json

# http status code
OK = 200
ACCEPTED = 202
INTERNAL_SERVER_ERROR= 500
BAD_REQUEST = 400

# the max retry time
RETRY_TIME = 10

BASE_URL = "http://127.0.0.1:8300/api/v1"

# we should write some SQLs in the run.sh after call create_changefeed
def create_changefeed():
    url = BASE_URL+"/changefeeds"
    # create changefeed
    for i in range(1, 4):
        data = json.dumps({
            "changefeed_id": "changefeed-test"+str(i),
            "sink_uri": "blackhole://",
            "ignore_ineligible_table": True
        })
        headers = {"Content-Type": "application/json"}
        resp = rq.post(url, data=data, headers=headers)
        assert resp.status_code == ACCEPTED

    # create changefeed fail
    data = json.dumps({
        "changefeed_id": "changefeed-test",
        "sink_uri": "mysql://127.0.0.1:1111",
        "ignore_ineligible_table": True
    })
    headers = {"Content-Type": "application/json"}
    resp = rq.post(url, data=data, headers=headers)
    assert resp.status_code == BAD_REQUEST
    print(resp.json())

    print("pass test: list changefeed")


def list_changefeed():
    # test state: all
    url = BASE_URL+"/changefeeds?state=all"
    resp = rq.get(url)
    assert resp.status_code == OK
    data = resp.json()
    assert len(data) == 3

    # test state: normal
    url = BASE_URL+"/changefeeds?state=normal"
    resp = rq.get(url)
    assert resp.status_code == OK
    data = resp.json()
    for changefeed in data:
        assert changefeed["state"] == "normal"

    # test state: stopped
    url = BASE_URL+"/changefeeds?state=stopped"
    resp = rq.get(url)
    assert resp.status_code == OK
    data = resp.json()
    for changefeed in data:
        assert changefeed["state"] == "stopped"

    print("pass test: list changefeed")

def get_changefeed():
    # test get changefeed success
    url = BASE_URL+"/changefeeds/changefeed-test1"
    resp = rq.get(url)
    assert resp.status_code == OK

    # test get changefeed failed
    url = BASE_URL+"/changefeeds/changefeed-not-exists"
    resp = rq.get(url)
    assert resp.status_code == BAD_REQUEST
    data = resp.json()
    assert data["error_code"] == "CDC:ErrChangeFeedNotExists"

    print("pass test: get changefeed")


def pause_changefeed():
    # pause changefeed
    url = BASE_URL+"/changefeeds/changefeed-test2/pause"
    resp = rq.post(url)
    assert resp.status_code == ACCEPTED

    # check if pause changefeed success
    url = BASE_URL+"/changefeeds/changefeed-test2"
    for i in range(RETRY_TIME):
        i += 1
        time.sleep(1)
        resp = rq.get(url)
        assert resp.status_code == OK
        data = resp.json()
        if data["state"] == "stopped":
            break
    assert data["state"] == "stopped"

    # test pause changefeed failed
    url = BASE_URL+"/changefeeds/changefeed-not-exists/pause"
    resp = rq.post(url)
    assert resp.status_code == BAD_REQUEST
    data = resp.json()
    assert data["error_code"] == "CDC:ErrChangeFeedNotExists"

    print("pass test: pause changefeed")

def update_changefeed():
    url = BASE_URL+"/changefeeds/changefeed-test1"


def resume_changefeed():
    # resume changefeed
    url = BASE_URL+"/changefeeds/changefeed-test2/resume"
    resp = rq.post(url)
    assert resp.status_code == ACCEPTED

    # check if resume changefeed success
    url = BASE_URL+"/changefeeds/changefeed-test2"
    for i in range(RETRY_TIME):
        i += 1
        time.sleep(1)
        resp = rq.get(url)
        assert resp.status_code == OK
        data = resp.json()
        if data["state"] == "normal":
            break
    assert data["state"] == "normal"

    # test resume changefeed failed
    url = BASE_URL+"/changefeeds/changefeed-not-exists/resume"
    resp = rq.post(url)
    assert resp.status_code == BAD_REQUEST
    data = resp.json()
    assert data["error_code"] == "CDC:ErrChangeFeedNotExists"

    print("pass test: resume changefeed")


def remove_changefeed():
    # remove changefeed
    url = BASE_URL+"/changefeeds/changefeed-test3"
    resp = rq.delete(url)
    assert resp.status_code == ACCEPTED

    # check if remove changefeed success
    url = BASE_URL+"/changefeeds/changefeed-test3"
    for i in range(RETRY_TIME):
        i += 1
        time.sleep(1)
        resp = rq.get(url)
        if resp.status_code == BAD_REQUEST:
            break

    assert resp.status_code == BAD_REQUEST
    assert resp.json()["error_code"] == "CDC:ErrChangeFeedNotExists"

    # test remove changefeed failed
    url = BASE_URL+"/changefeeds/changefeed-not-exists"
    resp = rq.delete(url)
    assert (resp.status_code == BAD_REQUEST or resp.status_code == INTERNAL_SERVER_ERROR)
    data = resp.json()
    assert data["error_code"] == "CDC:ErrChangeFeedNotExists"

    print("pass test: remove changefeed")


def rebalance_table():
    # rebalance_table
    url = BASE_URL + "/changefeeds/changefeed-test1/tables/rebalance_table"
    resp = rq.post(url)
    assert resp.status_code == ACCEPTED

    print("pass test: rebalance table")


def move_table():
    # move table
    url = BASE_URL + "/changefeeds/changefeed-test1/tables/move_table"
    data = json.dumps({"capture_id": "test-aaa-aa", "table_id": 11})
    headers = {"Content-Type": "application/json"}
    resp = rq.post(url, data=data, headers=headers)
    assert resp.status_code == ACCEPTED

    # move table fail
    # not allow empty capture_id
    data = json.dumps({"capture_id": "", "table_id": 11})
    headers = {"Content-Type": "application/json"}
    resp = rq.post(url, data=data, headers=headers)
    assert resp.status_code == BAD_REQUEST

    print("pass test: move table")


def resign_owner():
    url = BASE_URL + "/owner/resign"
    resp = rq.post(url)
    assert resp.status_code == ACCEPTED

    print("pass test: resign owner")


def list_capture():
    url = BASE_URL + "/captures"
    resp = rq.get(url)
    assert resp.status_code == OK

    print("pass test: list captures")


def list_processor():
    url = BASE_URL + "/processors"
    resp = rq.get(url)
    assert resp.status_code == OK

    print("pass test: list processors")


# must at least one table is sync will the test success
def get_processor():
    url = BASE_URL + "/processors"
    resp = rq.get(url)
    assert resp.status_code == OK
    data = resp.json()[0]
    url = url + "/" + data["changefeed_id"] + "/" + data["capture_id"]
    resp = rq.get(url)
    assert resp.status_code == OK

    print("pass test: get processors")


def check_health():
    url = BASE_URL + "/health"
    resp = rq.get(url)
    assert resp.status_code == OK

    print("pass test: check health")


def get_status():
    url = BASE_URL + "/status"
    resp = rq.get(url)
    assert resp.status_code == OK
    assert  resp.json()["is_owner"] == True

    print("pass test: get status")


def set_log_level():
    url = BASE_URL + "/log"
    data = json.dumps({"log_level": "debug"})
    headers = {"Content-Type": "application/json"}
    resp = rq.post(url, data=data, headers=headers)
    assert resp.status_code == OK

    data = json.dumps({"log_level": "info"})
    resp = rq.post(url, data=data, headers=headers)
    assert resp.status_code == OK

    print("pass test: set log level")


if __name__ == "__main__":
    # test all the func as the order list in this map
    FUNC_MAP = {
        "check_health": check_health,
        "get_status": get_status,
        "create_changefeed": create_changefeed,
        "list_changefeed": list_changefeed,
        "get_changefeed": get_changefeed,
        "pause_changefeed": pause_changefeed,
        "resume_changefeed": resume_changefeed,
        "update_changefeed": update_changefeed,
        "rebalance_table": rebalance_table,
        "move_table": move_table,
        "get_processor": get_processor,
        "list_processor": list_processor,
        "set_log_level": set_log_level,
        "remove_changefeed": remove_changefeed,
        "resign_owner": resign_owner,
    }

    func = FUNC_MAP[sys.argv[1]]
    if len(sys.argv) >= 2:
        func(*sys.argv[2:])
    else:
        func()
