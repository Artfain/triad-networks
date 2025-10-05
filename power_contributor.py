import os
import subprocess
import sys
import time
import psutil
import websocket
import json
import threading
import socket

def update_pip():
    print("Checking pip version...")
    subprocess.check_call([sys.executable, "-m", "pip", "install", "--upgrade", "pip"])

def get_cpu_temp():
    try:
        result = subprocess.run(['osx-cpu-temp'], capture_output=True, text=True)
        temp_str = result.stdout.strip().split()[0]
        return float(temp_str.replace('°C', ''))
    except Exception as e:
        print(f"Error getting CPU temp: {e}")
        return 0.0

def check_server_connection(host="localhost", port=8080):
    attempts = 5
    for i in range(attempts):
        try:
            sock = socket.create_connection((host, port), timeout=2)
            sock.close()
            return True
        except socket.error as e:
            print(f"Attempt {i+1}/{attempts}: Server not available at {host}:{port}, retrying... ({e})")
            time.sleep(2)
    return False

def contribute_power(ws, cpu_load, mfa_token, address="user1", device_id="macbook"):
    while True:
        cpu_percent = psutil.cpu_percent(interval=1)
        memory = psutil.virtual_memory()
        memory_mb = memory.used / (1024 * 1024)
        storage = psutil.disk_usage('/').used / (1024 * 1024 * 1024)
        bandwidth = psutil.net_io_counters().bytes_sent / (1024 * 1024)
        uptime = int(time.time() - psutil.boot_time())
        eco_actions = 1
        cpu_temp = get_cpu_temp()
        if cpu_temp > 85:
            print("CPU temperature too high, stopping contribution")
            break
        data = {
            "action": "contributePower",
            "userData": {"address": address},
            "deviceID": device_id,
            "power": {"cpuPercent": cpu_percent, "memoryMB": memory_mb},
            "storage": storage,
            "bandwidth": bandwidth,
            "uptime": uptime,
            "ecoActions": eco_actions,
            "mfaToken": mfa_token
        }
        try:
            ws.send(json.dumps(data))
            print(f"Contributed: CPU {cpu_percent:.1f}%, Memory {memory_mb:.2f} MB, Storage {storage:.0f} GB, Bandwidth {bandwidth:.0f} MB, Uptime {uptime} sec, EcoActions {eco_actions}")
        except Exception as e:
            print(f"Error sending contribution: {e}")
            break
        time.sleep(10)

def on_message(ws, message):
    global mfa_token
    data = json.loads(message)
    print(f"Received: {data}")
    if "mfaToken" in data:
        mfa_token = data["mfaToken"]
        print(f"MFA token updated: {mfa_token}")

def on_error(ws, error):
    print(f"Error: {error}")

def on_close(ws, close_status_code, close_msg):
    print(f"Closed: {close_msg}")

def on_open(ws):
    print("Connected to server")
    data = {
        "action": "addUser",
        "userData": {
            "address": "user1",
            "balance": 1000,
            "devices": ["macbook"],
            "pocContribution": {
                "computations": 0,
                "storage": 0,
                "bandwidth": 0,
                "uptime": 0,
                "ecoActions": 0
            }
        },
        "deviceID": "macbook"
    }
    ws.send(json.dumps(data))

if __name__ == "__main__":
    update_pip()
    if not check_server_connection():
        print("Cannot connect to server, exiting...")
        sys.exit(1)
    websocket.enableTrace(True)
    ws = websocket.WebSocketApp("ws://localhost:8080/ws",
                                on_message=on_message,
                                on_error=on_error,
                                on_close=on_close,
                                on_open=on_open)
    global mfa_token
    mfa_token = ""
    cpu_load = float(sys.argv[1]) if len(sys.argv) > 1 else 50.0
    ws_thread = threading.Thread(target=ws.run_forever)
    ws_thread.start()
    time.sleep(2)
    while not mfa_token:
        time.sleep(1)
    print("User added and MFA token received, starting to contribute power")
    contribute_power(ws, cpu_load, mfa_token)