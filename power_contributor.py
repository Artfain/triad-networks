import websocket
import json
import time
import sys
import psutil
import hashlib
import ecdsa
import base64

def generate_keypair():
    sk = ecdsa.SigningKey.generate(curve=ecdsa.SECP256k1)
    vk = sk.get_verifying_key()
    return sk.to_string().hex(), vk.to_string().hex()

def sign_message(private_key, message):
    sk = ecdsa.SigningKey.from_string(bytes.fromhex(private_key), curve=ecdsa.SECP256k1)
    message_hash = hashlib.sha256(message.encode()).digest()
    signature = sk.sign(message_hash)
    return base64.b64encode(signature).decode()

def contribute_power(address, device_id, cpu_load):
    ws = websocket.WebSocket()
    ws.connect("ws://localhost:8080/ws")
    
    # Register user
    register_msg = {
        "type": "register",
        "data": {
            "address": address,
            "deviceID": device_id
        }
    }
    ws.send(json.dumps(register_msg))
    response = json.loads(ws.recv())
    print(f"Registration response: {response}")

    # Simulate contribution
    while True:
        computations = int(cpu_load * 1000)
        eco_actions = int(cpu_load / 10)
        trees_planted = eco_actions // 5
        contribution = {
            "computations": computations,
            "storage": psutil.disk_usage('/').free / (1024**3),  # Free disk space in GB
            "bandwidth": 100.0,  # Simulated bandwidth in Mbps
            "uptime": int(time.time()),
            "ecoActions": eco_actions
        }
        contribute_msg = {
            "type": "contribute",
            "data": {
                "address": address,
                "deviceID": device_id,
                "contribution": contribution,
                "trees": trees_planted
            }
        }
        ws.send(json.dumps(contribute_msg))
        response = json.loads(ws.recv())
        print(f"Contribution response: {response}")

        # Request user data
        data_msg = {
            "type": "get_data",
            "data": {"address": address}
        }
        ws.send(json.dumps(data_msg))
        user_data = json.loads(ws.recv())
        print(f"User data: {user_data}")

        # Request transactions
        tx_msg = {
            "type": "get_transactions",
            "data": {"address": address}
        }
        ws.send(json.dumps(tx_msg))
        transactions = json.loads(ws.recv())
        print(f"Transactions: {transactions}")

        # Request trees planted
        trees_msg = {
            "type": "get_trees",
            "data": {"address": address}
        }
        ws.send(json.dumps(trees_msg))
        trees_data = json.loads(ws.recv())
        print(f"Trees planted: {trees_data}")

        time.sleep(60)  # Contribute every minute

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python power_contributor.py <cpu_load>")
        sys.exit(1)
    
    cpu_load = float(sys.argv[1])
    private_key, public_key = generate_keypair()
    address = hashlib.sha256(public_key.encode()).hexdigest()[:40]
    device_id = "macbook"
    
    try:
        contribute_power(address, device_id, cpu_load)
    except KeyboardInterrupt:
        print("Stopped contributing")