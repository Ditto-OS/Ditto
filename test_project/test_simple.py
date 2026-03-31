# Simple test
import requests

resp = requests.get("http://test.com")
print(resp)
print(resp.status_code)
