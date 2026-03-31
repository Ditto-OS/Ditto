# Debug test
import requests
print("Module loaded:", requests)

resp = requests.get("http://test.com")
print("Response:", resp)
print("Type:", type(resp))
