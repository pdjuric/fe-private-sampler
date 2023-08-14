import requests
import time
import json

server_ip = {
    "schema": "http",
    "ipv4": "192.168.100.43",
    "port": "8080"
}

sensor_ip = {
    "schema": "http",
    "ipv4": "192.168.100.43",
    "port": "8081"
}

def get_base_url(ip):
    return ip['schema'] + '://' + ip['ipv4'] + ':' + ip['port']


def POST(url, json):
    response = requests.post(url, json=json)
    print(response.status_code)
    if response.content:
        print(response.text)

    return response.status_code // 100 == 2


def set_server_to_sensor(server_ip, sensor_ip):
    print('>>> set server to sensor')
    url = get_base_url(sensor_ip) + '/server'
    return POST(url, server_ip)


def create_group(server_ip):
    print('>>> creating group')
    response = requests.post(get_base_url(server_ip) + "/group")
    print(response.status_code)
    if response.content:
        print(response.text)

    if response.status_code // 100 != 2:
        return None

    return response.json()['id']


def set_group_to_sensor(sensor_ip, group_id):
    json = {
        "id": group_id
    }

    response = requests.post(get_base_url(sensor_ip) + '/group', json=json)
    print(response.status_code)
    if response.content:
        print(response.text)

    return response.status_code // 100 == 2


def register_sensor_to_server(sensor_ip):
    print('>>> registering sensor')
    response = requests.get(get_base_url(sensor_ip) + '/register')
    print(response.status_code)
    if response.content:
        print(response.text)

    return response.status_code // 100 == 2


def add_task(server_ip, taskDef):
    print('>>> adding task')
    response = requests.post(get_base_url(server_ip) + "/task", json=taskDef)
    print(response.status_code)
    if response.content:
        print(response.text)

    if response.status_code // 100 != 2:
        return None

    return response.text


def get_task_info_server(server_ip, taskId):
    response = requests.get(get_base_url(server_ip) + "/task/" + taskId)
    print(response.status_code)
    if response.content:
        print(json.dumps(json.loads(response.text), indent=4))

    return json.loads(response.text)

def get_task_samples_sensor(sensor_ip, taskId):
    response = requests.get(get_base_url(sensor_ip) + "/task/" + taskId + '/samples')
    print(response.status_code)
    if response.content:
        print(response.text)


    import ast
    return ast.literal_eval(response.text)


def calc_result_for_sensor(taskDef, samples):
    batchSize = taskDef['batchSize']
    batchCnt = taskDef['sampleCount'] // batchSize

    res = 0

    i, j =  0, 0
    while batchCnt != i:
        res += samples[i][j] * taskDef['coefficientsByPeriod'][i*batchCnt + j]

        j += 1
        if j == batchSize:
            j = 0
            i += 1

    return res


def add_task_batch(taskDef, taskCnt):
    i = 0
    tasks = []
    while i < taskCnt:
        tasks = tasks + [add_task(server_ip, taskDef)]
        i += 1

    return tasks



def check_result(taskDef, taskId):
    samples = get_task_samples_sensor(sensor_ip, taskId)
    calculated = get_task_info_server(server_ip, taskId)['result']
    result = calc_result_for_sensor(taskDef, samples)
    return calculated == result, samples, calculated, result


def init():
    set_server_to_sensor(server_ip, sensor_ip)
    group_id = create_group(server_ip)

    set_group_to_sensor(sensor_ip, group_id)
    register_sensor_to_server(sensor_ip)

    return server_ip, sensor_ip, group_id



def get_task_def_multi_1():
    low = 20840
    high = 83360
    day = [high]*7 + [low]*15 + [high]*2
    return {
        "group_id": group_id,
        "start": int(time.time()) + 5,
        "samplingPeriod": 3,
        "batchSize": 24,
        "sampleCount": 24*2,
        "maxSampleValue": 200,
        "coefficientsByPeriod": day * 2
    }


def get_task_def_simple_1():
    t = [1] * 2
    g = [1] * 2
    return {
        "group_id": group_id,
        "start": int(time.time()) + 5,
        "samplingPeriod": 2,
        "batchSize": 4,
        "sampleCount": 2*2,
        "maxSampleValue": 10,
        "coefficientsByPeriod": t + g
    }


server_ip, sensor_ip, group_id = init()
