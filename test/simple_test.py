#!/usr/bin/env python3
import requests
import time


serverUrl = 'http://192.168.100.6:8080'
sensorUrl = 'http://192.168.100.6:8081'

def main():
    # set server to sensor
    print('>>> set server to sensor')
    json = {
        "schema": "http",
        "ipv4": "192.168.100.6",
        "port": "8080"
    }

    response = requests.post(sensorUrl + '/server', json=json)
    print(response.status_code)
    if response.content:
        print(response.text)

    if response.status_code // 100 != 2:
        return


    # create group
    print('>>> creating group')
    response = requests.post(serverUrl + "/group")
    print(response.status_code)
    if response.content:
        print(response.text)

    if response.status_code // 100 != 2:
        return

    group_id = response.json()['id']

    # set group to sensor
    print('>>> setting group to sensor')
    json = {
        "id": group_id
    }

    response = requests.post(sensorUrl + '/group', json=json)
    print(response.status_code)
    if response.content:
        print(response.text)


    # register sensor
    print('>>> registering sensor')
    response = requests.get(sensorUrl + '/register')
    print(response.status_code)
    if response.content:
        print(response.text)
    if response.status_code // 100 != 2:
        return


    # add task
    print('>>> adding task')
    json = {
        "groupId": group_id,
        "start": int(time.time()) + 5,
        "samplingPeriod": 3,
        "batchSize": 6,
        "sampleCount": 6,
        "maxSampleValue": 30,
        "coefficientsByPeriod": [1, 2, 3, 4, 5, 6]
    }


    response = requests.post(serverUrl + "/task", json=json)
    print(response.status_code)
    if response.content:
        print(response.text)

    if response.status_code // 100 != 2:
        return


main()
