import requests
import time
import json


class Tariff:

    def __init__(self, description, sampling_period, batch_size, max_sample_value, max_tariff_value):
        self.id = None
        self.description = description
        self.sampling_period = sampling_period
        self.batch_size = batch_size
        self.max_sample_value = max_sample_value
        self.max_tariff_value = max_tariff_value

    def json(self):
        return {
            "description": self.description,
            "samplingPeriod": self.sampling_period,
            "batchSize": self.batch_size,
            "maxSampleValue": self.max_sample_value,
            "maxTariffValue": self.max_tariff_value
        }


class Task:

    def __init__(self, customer_id, start, batch_cnt, tariff: Tariff, encrypt):
        self.id = None
        self.details_from_server = None
        self.samples = None
        self.customer_id = customer_id
        self.start = start
        self.encrypt = encrypt
        self.duration = tariff.sampling_period * tariff.batch_size * batch_cnt
        self.tariff = tariff

    def json(self):
        return {
            "customerId": self.customer_id,
            "start": self.start,
            "duration": self.duration,
            "tariffId": self.tariff.id,
            "enableEncryption": self.encrypt
        }




class HttpServer:
    def __init__(self, schema, ip, port):
        self.schema = schema
        self.ip = ip
        self.port = port

    def get_base_url(self):
        return self.schema + '://' + self.ip + ':' + self.port

    def get_ip(self):
        return {
            "schema": self.schema,
            "ipv4": self.ip,
            "port": self.port
        }

    def POST(self, url, json=None):
        if json:
            response = requests.post(self.get_base_url() + url, json=json)
        else:
            response = requests.post(self.get_base_url() + url)
        print(response.status_code)
        if response.content:
            print(response.text)

        return response.status_code // 100 == 2, response.text

    def GET(self, url):
        response = requests.get(self.get_base_url() + url)
        print(response.status_code)
        if response.content:
            print(response.text)

        return response.status_code // 100 == 2, response.text

class Authority(HttpServer):
    pass

class Server(HttpServer):
    def create_customer(self):
        print('>>> creating customer')
        code, customer_id = self.POST('/customer')
        return json.loads(customer_id)['id']

    def set_authority(self, authority: Authority):
        print('>>> setting authority')
        self.POST('/authority', authority.get_ip())

    def add_task(self, task: Task):
        print('>>> adding task')
        code, task_id = self.POST('/task', task.json())
        task.id = task_id

    def get_task_info(self, task: Task):
        code, body = self.GET(f'/task/{task.id}')
        task.details_from_server += [json.loads(body)]
        print(json.dumps(task.details_from_server, indent=4))

    def add_tariff(self, tariff: Tariff):
        code, body = self.POST('/tariff', tariff.json())
        tariff.id = body


class Sensor(HttpServer):

    def set_server(self, server: Server):
        print('>>> set server to sensor')
        self.POST('/server', server.get_ip())

    def set_customer(self, customer_id):
        json = {
            "id": customer_id
        }
        self.POST('/customer', json)

    def register(self):
        print('>>> registering sensor')
        self.GET('/register')

    def get_task_info(self, task: Task):
        code, body = self.GET(f'/task/{task.id}')
        task.details_from_server = json.loads(body)
        print(json.dumps(task.details_from_server, indent=4))

    def get_task_samples(self, task: Task):
        code, body = self.GET(f'/task/{task.id}/samples')
        import ast
        task.samples = ast.literal_eval(body)


class TestUtil:

    def __init__(self, server, sensors, authority):
        self.sensors = sensors
        self.server = server
        self.authority = authority
        self.customerId = None

        self.init()

    def __init__(self):
        self.server = Server("http", "127.0.0.1", "8080")
        self.sensor = Sensor("http", "127.0.0.1", "8081")
        self.authority = Authority("http", "127.0.0.1", "8082")


    def init(self):
        self.sensor.set_server(self.server)

        customer_id = self.server.create_customer()
        self.sensor.set_customer(customer_id)
        self.sensor.register()
        self.server.set_authority(self.authority)

        self.customer_id = customer_id




# def calc_result_for_sensor(taskDef, samples):
#     batchSize = taskDef['batchSize']
#     batchCnt = taskDef['sampleCount'] // batchSize
#
#     res = 0
#
#     i, j = 0, 0
#     while batchCnt != i:
#         res += samples[i][j] * taskDef['coefficientsByPeriod'][i * batchCnt + j]
#
#         j += 1
#         if j == batchSize:
#             j = 0
#             i += 1
#
#     return res


# def add_task_batch(taskDef, taskCnt):
#     i = 0
#     tasks = []
#     while i < taskCnt:
#         tasks = tasks + [add_task(server_ip, taskDef)]
#         i += 1
#
#     return tasks


# def check_result(taskDef, taskId):
#     samples = get_task_samples_sensor(sensor_ip, taskId)
#     calculated = get_task_info_server(server_ip, taskId)['result']
#     result = calc_result_for_sensor(taskDef, samples)
#     return calculated == result, samples, calculated, result


# max 10000 kWh/month -> 3.47722 kWh/15min -> scale x100 for better precision
# max rate 20 -> scale 10_000x
def one_month_tariff_benchmark():
    test_util = TestUtil()
    server = test_util.server
    customer_id = test_util.customer_id

    one_month_tariff = Tariff("tariff_1", 1, 8, 300, 200_000)
    server.add_tariff(one_month_tariff)

    task = Task(customer_id, int(time.time()) + 5, 4, one_month_tariff, True)
    server.add_task(one_month_tariff)

def single_benchmark():
    test_util = TestUtil()
    server = test_util.server
    customer_id = test_util.customer_id

    single_tariff = Tariff("single_tariff", 1, 40, 300, 200000)
    server.add_tariff(single_tariff)

#     w/o encryption
#     task = Task(customer_id, int(time.time()) + 5, 1, single_tariff, False)
#     server.add_task(single_tariff)

    # w/ encryption
    task = Task(customer_id, int(time.time()) + 5, 1, tariff_1, True)
    server.add_task(task)

def multi_benchmark():
    test_util = TestUtil()
    server = test_util.server
    customer_id = test_util.customer_id

    multi_tariff = Tariff("multi_tariff", 1, 20, 300, 200000)
    server.add_tariff(multi_tariff)

    task = Task(customer_id, int(time.time()) + 5, 10, multi_tariff, True)
    server.add_task(task)
    return task

def single_vs_multi_benchmark():
    test_util = TestUtil()
    server = test_util.server
    customer_id = test_util.customer_id

    single_tariff = Tariff("single_tariff", 1, 60, 300, 200000)
    server.add_tariff(single_tariff)

    # w/ encryption
    single_task = Task(customer_id, int(time.time()) + 15, 1, single_tariff, True)
    server.add_task(single_task)

    # wait for completion
    time.sleep(80)

    # w/o encryption
    single_task_no_encryption = Task(customer_id, int(time.time()) + 15, 1, single_tariff, False)
    server.add_task(single_task_no_encryption)

    # wait for completion
    time.sleep(80)

    multi_tariff = Tariff("multi_tariff", 1, 10, 300, 200000)
    server.add_tariff(multi_tariff)

    # w/ encryption
    multi_task = Task(customer_id, int(time.time()) + 15, 6, multi_tariff, True)
    server.add_task(multi_task)

    # w/ encryption
    time.sleep(80)

    # w/o encryption
    multi_task_no_encryption = Task(customer_id, int(time.time()) + 15, 6, multi_tariff, False)
    server.add_task(multi_task_no_encryption)


singleVsMulti()
