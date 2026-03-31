# Test real package usage
import requests

print("Testing requests package...")

# Test GET request
resp = requests.get("https://api.example.com/data")
print("GET status:", resp.status_code)
print("GET response:", resp.text)

# Test POST request  
resp = requests.post("https://api.example.com/users", json={"name": "Alice"})
print("POST status:", resp.status_code)
print("POST response:", resp.text)

# Test json method
resp = requests.get("https://api.example.com/json")
data = resp.json()
print("JSON data:", data)

print("All requests tests passed!")
